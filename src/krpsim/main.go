package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"krpsim/src/system/cmd"
	"krpsim/src/system/models"
	"krpsim/src/system/parsing"
)

// ! getOptCount returns the number of optimization targets defined in configuration
func getOptCount(cfg *models.Config) int {
	if len(cfg.Targets) > 0 {
		return len(cfg.Targets)
	}
	if cfg.OptimizeTime {
		return 1
	}
	return 0
}

// * displays the krpsim simulation results formatted
func printOutput(cfg *models.Config, res cmd.SimResult) {
	optCount := getOptCount(cfg)

	fmt.Printf("> 【=◈︿◈=】= %d processes, %d stocks, %d to optimize\n", len(cfg.Processes), len(cfg.StockNames), optCount)
	fmt.Println("Evaluating .................. done.")
	fmt.Println("【=◈︿◈=】")

	for _, step := range res.Steps {
		fmt.Printf("%d:%s\n", step.Cycle, step.Proc)
	}

	fmt.Printf("no more process doable at time %d\n", res.LastCycle+1)
	fmt.Println("Stock :")

	for i, name := range cfg.StockNames {
		qty := 0
		if i < len(res.Stocks) {
			qty = res.Stocks[i]
		}
		fmt.Printf(" %s => %d\n", name, qty)
	}
}

func main() {
	// * check the number of arguments
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: krpsim <file> <delay>")
		os.Exit(1)
	}

	delay, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil || delay <= 0 {
		fmt.Fprintf(os.Stderr, "error: invalid delay %q: must be positive number\n", os.Args[2])
		os.Exit(1)
	}

	cfg, err := parsing.ParseFile(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: parse config %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}

	timeout := time.Duration(delay * 0.9 * float64(time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	res := cmd.RunGreedy(ctx, cfg, 0)
	printOutput(cfg, res)
}
