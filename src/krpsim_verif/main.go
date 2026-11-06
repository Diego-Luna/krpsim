package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"krpsim/src/system/engine"
	"krpsim/src/system/models"
	"krpsim/src/system/parsing"
)

// * VerifyResult stores the outcome of replaying a krpsim trace
type VerifyResult struct {
	OK        bool
	ErrCycle  int
	ErrProc   string
	ErrReason string
	LastCycle int
	Stocks    []int
}

// * isDisplayLine checks if a trimmed trace line is a krpsim banner or stock summary line
func isDisplayLine(line string) bool {
	if strings.Contains(line, "=>") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "【") {
		return true
	}
	lower := strings.ToLower(line)
	for _, prefix := range []string{"file", "evaluating", "main walk", "no more process"} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return lower == "stock :" || lower == "stock:"
}

// ! parseTraceLine extracts (cycle, procName) or returns skip=true for comments/banners
func parseTraceLine(rawLine string) (int, string, bool, error) {
	line := strings.TrimSpace(strings.SplitN(rawLine, "#", 2)[0])
	if line == "" || isDisplayLine(line) {
		return 0, "", true, nil
	}
	colonIdx := strings.IndexByte(line, ':')
	if colonIdx <= 0 || colonIdx == len(line)-1 {
		return 0, "", false, fmt.Errorf("malformed trace line %q: expected <cycle>:<process>", line)
	}
	cycleStr := strings.TrimSpace(line[:colonIdx])
	procName := strings.TrimSpace(line[colonIdx+1:])
	if procName == "" {
		return 0, "", false, fmt.Errorf("malformed trace line %q: empty process name", line)
	}
	cycle, err := strconv.Atoi(cycleStr)
	if err != nil || cycle < 0 {
		return 0, procName, false, fmt.Errorf("invalid cycle %q in trace line %q", cycleStr, line)
	}
	return cycle, procName, false, nil
}

// * verifyTrace replays the trace against cfg using the discrete-event engine
func verifyTrace(cfg *models.Config, r io.Reader) (VerifyResult, error) {
	state := engine.NewState(cfg)
	fail := func(c int, p, reason string) (VerifyResult, error) {
		return VerifyResult{ErrCycle: c, ErrProc: p, ErrReason: reason, LastCycle: state.Cycle, Stocks: append([]int(nil), state.Stocks...)}, nil
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		cycle, procName, skip, err := parseTraceLine(scanner.Text())
		if err != nil {
			return fail(state.Cycle, procName, err.Error())
		}
		if skip {
			continue
		}
		if cycle < state.Cycle {
			return fail(cycle, procName, fmt.Sprintf("cycle %d is before current cycle %d", cycle, state.Cycle))
		}
		state.AdvanceTo(cycle)
		procIdx, ok := cfg.ProcIndex[procName]
		if !ok {
			return fail(cycle, procName, "unknown process")
		}
		if err := state.Launch(procIdx); err != nil {
			return fail(cycle, procName, err.Error())
		}
	}
	if err := scanner.Err(); err != nil {
		return VerifyResult{}, err
	}
	state.Drain()
	return VerifyResult{OK: true, LastCycle: state.Cycle, Stocks: append([]int(nil), state.Stocks...)}, nil
}

// * printVerifOutput formats and writes the verification verdict, last cycle, and final stocks
func printVerifOutput(w io.Writer, cfg *models.Config, res VerifyResult) {
	if res.OK {
		fmt.Fprintln(w, "Trace is valid!")
	} else {
		fmt.Fprintf(w, "Trace error at cycle %d, process %q: %s\n", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
	fmt.Fprintf(w, "Last cycle: %d\n", res.LastCycle)
	fmt.Fprintln(w, "Stock :")
	for i, name := range cfg.StockNames {
		qty := 0
		if i < len(res.Stocks) {
			qty = res.Stocks[i]
		}
		fmt.Fprintf(w, " %s => %d\n", name, qty)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: krpsim_verif <file> <result_to_test>")
		os.Exit(1)
	}
	cfg, err := parsing.ParseFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse config %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}
	var reader io.Reader = os.Stdin
	if os.Args[2] != "-" {
		f, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: open trace %q: %v\n", os.Args[2], err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	}
	res, err := verifyTrace(cfg, reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read trace: %v\n", err)
		os.Exit(1)
	}
	printVerifOutput(os.Stdout, cfg, res)
	if !res.OK {
		os.Exit(1)
	}
}
