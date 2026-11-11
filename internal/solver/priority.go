package solver

import (
	"sort"

	"krpsim/internal/config"
)

// rank orders the useful processes best first. A process is useful when it
// produces something that leads to an optimize target.
func rank(cfg *config.Config) (order []int, useful []bool) {
	value := make([]float64, len(cfg.StockNames))
	useful = make([]bool, len(cfg.Processes))

	if len(cfg.Targets) == 0 {
		for p := range useful {
			useful[p] = true
		}
		for s := range value {
			value[s] = neutralValue
		}
	} else {
		for _, t := range cfg.Targets {
			value[t] = targetValue
		}
		propagateValue(cfg, value, useful)
	}

	score := make([]float64, len(cfg.Processes))
	for p := range cfg.Processes {
		if useful[p] {
			score[p] = scoreOf(&cfg.Processes[p], value)
			order = append(order, p)
		}
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if score[a] != score[b] {
			return score[a] > score[b]
		}
		return a < b
	})
	return order, useful
}

// propagateValue walks value backwards from the targets: an input is worth
// what it helps produce, shrunk by valueDecay at every step.
func propagateValue(cfg *config.Config, value []float64, useful []bool) {
	for range valuePasses {
		for p := range cfg.Processes {
			proc := &cfg.Processes[p]
			for s, net := range proc.Net {
				if net <= 0 || value[s] <= 0 {
					continue
				}
				useful[p] = true
				for n, need := range proc.NeedQty {
					if need <= 0 {
						continue
					}
					if v := value[s] * float64(net) / (float64(need) * valueDecay); v > value[n] {
						value[n] = v
					}
				}
			}
		}
	}
}

func scoreOf(proc *config.Process, value []float64) float64 {
	gain, cost := 0.0, 0.0
	for _, r := range proc.Results {
		gain += value[r.Stock] * float64(r.Qty)
	}
	for _, n := range proc.Needs {
		cost += value[n.Stock] * float64(n.Qty)
	}
	return (gain - cost*inputPenalty) / float64(proc.Delay+1)
}
