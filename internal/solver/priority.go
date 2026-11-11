package solver

import (
	"sort"

	"krpsim/internal/config"
)

// ? maybe change the place for the models

// * ProcessDeltas holds per-stock needs, results, and net production for a process
type ProcessDeltas struct {
	Needs   []int
	Results []int
	Net     []int
}

// * PriorityResult holds the backward propagation outcome, useful processes, and execution order
type PriorityResult struct {
	Order  []int
	Useful []bool
	Deltas []ProcessDeltas
}

// Aggregates needs, results, and net per-stock changes for all processes
func ComputeProcessDeltas(cfg *config.Config) []ProcessDeltas {
	deltas := make([]ProcessDeltas, len(cfg.Processes))
	numStocks := len(cfg.StockNames)

	for pIdx := range cfg.Processes {
		proc := &cfg.Processes[pIdx]
		d := ProcessDeltas{
			Needs:   make([]int, numStocks),
			Results: make([]int, numStocks),
			Net:     make([]int, numStocks),
		}
		for _, need := range proc.Needs {
			if need.Stock >= 0 && need.Stock < numStocks {
				d.Needs[need.Stock] += need.Qty
			}
		}
		for _, res := range proc.Results {
			if res.Stock >= 0 && res.Stock < numStocks {
				d.Results[res.Stock] += res.Qty
			}
		}
		for s := 0; s < numStocks; s++ {
			d.Net[s] = d.Results[s] - d.Needs[s]
		}
		deltas[pIdx] = d
	}
	return deltas
}

// * runs backward propagation to propagate stock values and mark useful processes
func propagateStockValues(cfg *config.Config, deltas []ProcessDeltas, stockVal []float64, useful []bool) {
	for pass := 0; pass < 15; pass++ {
		for pIdx := range cfg.Processes {
			d := &deltas[pIdx]
			for s, netQty := range d.Net {
				if netQty <= 0 || stockVal[s] <= 0 {
					continue
				}
				useful[pIdx] = true
				for n, needQty := range d.Needs {
					if needQty <= 0 {
						continue
					}
					cand := (stockVal[s] * float64(netQty)) / (float64(needQty) * 1.5)
					if cand > stockVal[n] {
						stockVal[n] = cand
					}
				}
			}
		}
	}
}

// computes the priority score for a useful process
func scoreProcess(proc *config.Process, stockVal []float64) float64 {
	sumRes := 0.0
	for _, res := range proc.Results {
		if res.Stock >= 0 && res.Stock < len(stockVal) {
			sumRes += stockVal[res.Stock] * float64(res.Qty)
		}
	}
	sumNeed := 0.0
	for _, need := range proc.Needs {
		if need.Stock >= 0 && need.Stock < len(stockVal) {
			sumNeed += stockVal[need.Stock] * float64(need.Qty)
		}
	}
	num := sumRes - (sumNeed * 0.3)
	return num / float64(proc.Delay+1)
}

// ! ranks useful processes backwards from cfg.Targets
func ComputePriority(cfg *config.Config) PriorityResult {
	if cfg == nil {
		return PriorityResult{}
	}
	numStocks, numProcs := len(cfg.StockNames), len(cfg.Processes)
	stockVal, useful := make([]float64, numStocks), make([]bool, numProcs)
	deltas := ComputeProcessDeltas(cfg)

	if len(cfg.Targets) == 0 {
		for p := range useful {
			useful[p] = true
		}
		for s := range stockVal {
			stockVal[s] = 1.0
		}
	} else {
		for _, t := range cfg.Targets {
			if t >= 0 && t < numStocks {
				stockVal[t] = 1000.0
			}
		}
		propagateStockValues(cfg, deltas, stockVal, useful)
	}

	scores, order := make([]float64, numProcs), make([]int, 0, numProcs)
	for p := range cfg.Processes {
		if useful[p] {
			scores[p] = scoreProcess(&cfg.Processes[p], stockVal)
			order = append(order, p)
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		if scores[order[i]] != scores[order[j]] {
			return scores[order[i]] > scores[order[j]]
		}
		return order[i] < order[j]
	})
	return PriorityResult{Order: order, Useful: useful, Deltas: deltas}
}
