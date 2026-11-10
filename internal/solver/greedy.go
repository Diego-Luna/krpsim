package cmd

import (
	"context"

	"krpsim/src/system/engine"
	"krpsim/src/system/models"
)

// ? maybe change the place for the models

// * TraceStep records a single process launch at a specific cycle
type TraceStep struct {
	Cycle int
	Proc  string
}

// * SimResult encapsulates the output of a greedy simulation
type SimResult struct {
	Steps     []TraceStep
	LastCycle int
	Stocks    []int
	Exhausted bool
}

// returns a boolean slice indicating which stocks are targets
func buildTargetSet(cfg *models.Config) []bool {
	isTarget := make([]bool, len(cfg.StockNames))
	for _, t := range cfg.Targets {
		if t >= 0 && t < len(isTarget) {
			isTarget[t] = true
		}
	}
	return isTarget
}

// attempts to greedily launch runnable processes at current state cycle up to 1000 times
func tryLaunchCycle(cfg *models.Config, prio PriorityResult, state *engine.State, inFlight, demand []int, isTarget, isCat, isRootCat []bool, catReserve int, steps *[]TraceStep) {
	for launchCount := 0; launchCount < 1000; launchCount++ {
		launched := false
		state.InFlightStocks(inFlight)
		for _, p := range prio.Order {
			if state.MaxRuns(p, nil) <= 0 {
				continue
			}
			if isBlockedByCatalystGuard(p, cfg, prio.Deltas, state.Stocks, inFlight, demand, isCat, isRootCat, catReserve) {
				continue
			}
			if isProcessCapped(p, cfg, prio.Deltas, state.Stocks, inFlight, demand, isTarget, isRootCat, catReserve) {
				continue
			}
			if err := state.Launch(p); err != nil {
				continue
			}
			*steps = append(*steps, TraceStep{Cycle: state.Cycle, Proc: cfg.Processes[p].Name})
			launched = true
			break
		}
		if !launched {
			break
		}
	}
}

// executes the greedy krpsim solver respecting backward priorities and stock reservations
func RunGreedy(ctx context.Context, cfg *models.Config, maxCycle int) SimResult {
	if cfg == nil {
		return SimResult{Exhausted: true}
	}
	if maxCycle <= 0 {
		maxCycle = 10000
	}
	state := engine.NewState(cfg)
	prio := ComputePriority(cfg)
	isTarget := buildTargetSet(cfg)
	demand := computeStockDemand(cfg, prio.Useful, prio.Deltas)
	isCat, isRootCat, catReserve := computeCatalystThreshold(cfg, prio.Deltas)
	inFlight := make([]int, len(cfg.StockNames))
	var steps []TraceStep
	exhausted := false

	for {
		if ctx.Err() != nil || state.Cycle > maxCycle {
			break
		}
		tryLaunchCycle(cfg, prio, state, inFlight, demand, isTarget, isCat, isRootCat, catReserve, &steps)
		nextCycle, hasEvent := state.NextEvent()
		if !hasEvent {
			exhausted = true
			break
		}
		if nextCycle > maxCycle || ctx.Err() != nil {
			break
		}
		state.AdvanceTo(nextCycle)
	}
	state.Drain()
	finalStocks := make([]int, len(state.Stocks))
	copy(finalStocks, state.Stocks)
	return SimResult{Steps: steps, LastCycle: state.Cycle, Stocks: finalStocks, Exhausted: exhausted}
}
