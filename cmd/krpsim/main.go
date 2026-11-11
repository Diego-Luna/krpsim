package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"krpsim/internal/config"
	"krpsim/internal/report"
	"krpsim/internal/solver"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: krpsim <file> <delay>")
		os.Exit(1)
	}

	delay, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil || delay <= 0 {
		fmt.Fprintf(os.Stderr, "error: invalid delay %q: must be positive number\n", os.Args[2])
		os.Exit(1)
	}

	cfg, err := config.ParseFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse config %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	// leave 10% of the budget for printing
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(delay*0.9*float64(time.Second)))
	defer cancel()

	report.Run(os.Stdout, cfg, solver.Run(ctx, cfg, 0))
}
