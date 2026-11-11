package config

import (
	"fmt"
	"strconv"
	"strings"
)

// i splits a line on ':' at parenthesis depth 0
// and we returns an error if parentheses are unbalanced
func splitColonDepth0(s string) ([]string, error) {
	var parts []string
	start := 0
	depth := 0

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("unexpected closing parenthesis at column %d", i+1)
			}
		case ':':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("unclosed parenthesis")
	}

	parts = append(parts, strings.TrimSpace(s[start:]))
	return parts, nil
}

// * SplitProcess splits a process line on ':' at parenthesis depth 0 into 4 parts: (name, needs, results, delay)
func SplitProcess(line string) ([]string, error) {
	parts, err := splitColonDepth0(line)
	if err != nil {
		return nil, err
	}
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid process format: expected 4 colon-separated fields, got %d", len(parts))
	}
	return parts, nil
}

// * parseStockDeltas parses a stock delta list like "(need1:qty1;need2:qty2)", so empty string or "()" yields nil deltas
// * Duplicate stocks are merged/summed
func parseStockDeltas(s string, cfg *Config) ([]StockDelta, error) {
	if s == "" || s == "()" {
		return nil, nil
	}
	if !strings.HasPrefix(s, "(") || !strings.HasSuffix(s, ")") {
		return nil, fmt.Errorf("expected deltas enclosed in parentheses, got %q", s)
	}

	inner := strings.TrimSpace(s[1 : len(s)-1])
	if inner == "" {
		return nil, nil
	}

	entries := strings.Split(inner, ";")
	deltas := make([]StockDelta, 0, len(entries))
	seen := make(map[int]int)

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("empty entry in stock list")
		}
		delta, err := parseSingleDelta(entry, cfg)
		if err != nil {
			return nil, err
		}
		if idx, exists := seen[delta.Stock]; exists {
			deltas[idx].Qty += delta.Qty
		} else {
			seen[delta.Stock] = len(deltas)
			deltas = append(deltas, delta)
		}
	}
	return deltas, nil
}

// * parses a single "<stock>:<qty>" pair and auto-registers the stock
func parseSingleDelta(entry string, cfg *Config) (StockDelta, error) {
	sub := strings.Split(entry, ":")
	if len(sub) != 2 {
		return StockDelta{}, fmt.Errorf("invalid stock delta format %q, expected <stock>:<qty>", entry)
	}
	stockName := strings.TrimSpace(sub[0])
	qtyStr := strings.TrimSpace(sub[1])
	if stockName == "" {
		return StockDelta{}, fmt.Errorf("stock name cannot be empty")
	}

	qty, err := strconv.Atoi(qtyStr)
	if err != nil {
		return StockDelta{}, fmt.Errorf("invalid quantity %q for stock %q", qtyStr, stockName)
	}
	if qty <= 0 {
		return StockDelta{}, fmt.Errorf("quantity for stock %q must be greater than 0, got %d", stockName, qty)
	}

	stockID := getOrRegisterStock(cfg, stockName, 0)
	return StockDelta{Stock: stockID, Qty: qty}, nil
}

// * validates and constructs a Process from 4 colon-separated parts
func parseProcess(parts []string, cfg *Config) (Process, error) {
	name := parts[0]
	if name == "" {
		return Process{}, fmt.Errorf("process name cannot be empty")
	}
	if _, exists := cfg.ProcIndex[name]; exists {
		return Process{}, fmt.Errorf("duplicate process %s", name)
	}

	needs, err := parseStockDeltas(parts[1], cfg)
	if err != nil {
		return Process{}, fmt.Errorf("process %q needs: %w", name, err)
	}

	results, err := parseStockDeltas(parts[2], cfg)
	if err != nil {
		return Process{}, fmt.Errorf("process %q results: %w", name, err)
	}

	delay, err := strconv.Atoi(parts[3])
	if err != nil {
		return Process{}, fmt.Errorf("process %q invalid delay %q", name, parts[3])
	}
	if delay < 0 {
		return Process{}, fmt.Errorf("process %q delay must be >= 0, got %d", name, delay)
	}

	proc := Process{
		Name:    name,
		Needs:   needs,
		Results: results,
		Delay:   delay,
	}
	cfg.ProcIndex[name] = len(cfg.Processes)
	cfg.Processes = append(cfg.Processes, proc)
	return proc, nil
}
