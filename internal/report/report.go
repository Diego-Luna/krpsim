package report

import (
	"fmt"
	"io"

	"krpsim/internal/config"
	"krpsim/internal/solver"
	"krpsim/internal/trace"
	"krpsim/internal/verify"
)

func Run(w io.Writer, cfg *config.Config, res solver.SimResult) {
	fmt.Fprintf(w, "> %s= %d processes, %d stocks, %d to optimize\n",
		trace.Mascot, len(cfg.Processes), len(cfg.StockNames), optCount(cfg))
	fmt.Fprintln(w, trace.Evaluating)
	fmt.Fprintln(w, trace.Mascot)
	for _, step := range res.Steps {
		fmt.Fprintln(w, step)
	}
	fmt.Fprintf(w, "%s %d\n", trace.Finished, res.LastCycle+1)
	stocks(w, cfg, res.Stocks)
}

func Verify(w io.Writer, cfg *config.Config, res verify.Result) {
	if res.OK {
		fmt.Fprintln(w, "Trace is valid!")
	} else {
		fmt.Fprintf(w, "Trace error at cycle %d, process %q: %s\n", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
	fmt.Fprintf(w, "Last cycle: %d\n", res.LastCycle)
	stocks(w, cfg, res.Stocks)
}

func stocks(w io.Writer, cfg *config.Config, qty []int) {
	fmt.Fprintln(w, trace.StockHeader)
	for i, name := range cfg.StockNames {
		n := 0
		if i < len(qty) {
			n = qty[i]
		}
		fmt.Fprintf(w, " %s %s %d\n", name, trace.StockArrow, n)
	}
}

func optCount(cfg *config.Config) int {
	if len(cfg.Targets) > 0 {
		return len(cfg.Targets)
	}
	if cfg.OptimizeTime {
		return 1
	}
	return 0
}
