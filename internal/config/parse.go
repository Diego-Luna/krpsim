package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func ParseFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file %q: %w", path, err)
	}
	defer f.Close()

	return Parse(f)
}

// Parse reads stock lines, process lines and one optimize line; # starts a comment.
func Parse(r io.Reader) (*Config, error) {
	cfg := &Config{
		StockIndex: make(map[string]int),
		ProcIndex:  make(map[string]int),
	}
	declaredStocks := make(map[string]bool)
	hasOptimize := false
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(strings.Split(scanner.Text(), "#")[0])
		if line == "" {
			continue
		}
		if err := parseLine(line, lineNum, cfg, declaredStocks, &hasOptimize); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(cfg.Processes) == 0 {
		return nil, fmt.Errorf("no processes defined")
	}
	if !hasOptimize {
		return nil, fmt.Errorf("missing optimize directive")
	}
	cfg.finalize()
	return cfg, nil
}

func parseLine(line string, lineNum int, cfg *Config, declaredStocks map[string]bool, hasOptimize *bool) error {
	parts, splitErr := splitTopLevel(line)
	if splitErr != nil {
		return fmt.Errorf("line %d: %w", lineNum, splitErr)
	}

	// with 4 fields it is a process that happens to be called "optimize"
	if parts[0] == "optimize" && len(parts) != 4 {
		if *hasOptimize {
			return fmt.Errorf("line %d: duplicate optimize directive", lineNum)
		}
		if len(parts) != 2 {
			return fmt.Errorf("line %d: invalid optimize directive format, expected optimize:(...)", lineNum)
		}
		if err := parseOptimize(parts[1], lineNum, cfg); err != nil {
			return err
		}
		*hasOptimize = true
		return nil
	}

	if len(parts) == 4 {
		if err := parseProcess(parts, cfg); err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		return nil
	}

	if len(parts) == 2 {
		return parseStock(parts[0], parts[1], lineNum, cfg, declaredStocks)
	}

	return fmt.Errorf("line %d: invalid line format: expected 2 or 4 colon-separated fields, got %d", lineNum, len(parts))
}

func parseStock(name, qtyStr string, lineNum int, cfg *Config, declaredStocks map[string]bool) error {
	if name == "" {
		return fmt.Errorf("line %d: stock name cannot be empty", lineNum)
	}
	if declaredStocks[name] {
		return fmt.Errorf("line %d: duplicate stock %s", lineNum, name)
	}
	qty, err := strconv.Atoi(qtyStr)
	if err != nil || qty < 0 {
		return fmt.Errorf("line %d: invalid stock quantity %q", lineNum, qtyStr)
	}
	declaredStocks[name] = true
	id := getOrRegisterStock(cfg, name, qty)
	cfg.InitialStocks[id] = qty
	return nil
}

// parseOptimize reads "(time;stock1;...)".
func parseOptimize(targetStr string, lineNum int, cfg *Config) error {
	if !strings.HasPrefix(targetStr, "(") || !strings.HasSuffix(targetStr, ")") {
		return fmt.Errorf("line %d: invalid optimize directive format, expected optimize:(...)", lineNum)
	}
	inner := strings.TrimSpace(targetStr[1 : len(targetStr)-1])
	if inner == "" {
		return fmt.Errorf("line %d: empty optimize directive", lineNum)
	}

	targetSet := make(map[int]bool)
	for _, id := range cfg.Targets {
		targetSet[id] = true
	}

	items := strings.Split(inner, ";")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			return fmt.Errorf("line %d: empty item in optimize directive", lineNum)
		}
		if item == "time" {
			cfg.OptimizeTime = true
			continue
		}
		stockID := getOrRegisterStock(cfg, item, 0)
		if !targetSet[stockID] {
			targetSet[stockID] = true
			cfg.Targets = append(cfg.Targets, stockID)
		}
	}
	return nil
}

// getOrRegisterStock gives every stock an id on first sight; stocks that only
// appear in processes or the optimize line start at initialQty (0).
func getOrRegisterStock(cfg *Config, name string, initialQty int) int {
	if id, exists := cfg.StockIndex[name]; exists {
		return id
	}
	id := len(cfg.StockNames)
	cfg.StockNames = append(cfg.StockNames, name)
	cfg.StockIndex[name] = id
	cfg.InitialStocks = append(cfg.InitialStocks, initialQty)
	return id
}
