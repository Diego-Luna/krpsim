package cmd

import (
	"krpsim/src/system/models"
)

// is process p is a slow non-catalyst producer
// of an output that another faster catalyst-based process also produces
func hasFastCatalystProducer(p int, cfg *models.Config, deltas []ProcessDeltas) bool {
	d := &deltas[p]
	for s, needQty := range d.Needs {
		if needQty > 0 && d.Net[s] == 0 {
			return false
		}
	}
	for outS, netOut := range d.Net {
		if netOut <= 0 {
			continue
		}
		for pOther := range cfg.Processes {
			if pOther == p || deltas[pOther].Net[outS] <= 0 {
				continue
			}
			for s, needQty := range deltas[pOther].Needs {
				if needQty > 0 && deltas[pOther].Net[s] == 0 && cfg.Processes[pOther].Delay < cfg.Processes[p].Delay {
					return true
				}
			}
		}
	}
	return false
}

// ! calculates maximum quantity needed per stock by any useful process
func computeStockDemand(cfg *models.Config, useful []bool, deltas []ProcessDeltas) []int {
	demand := make([]int, len(cfg.StockNames))
	for p, isU := range useful {
		if !isU || hasFastCatalystProducer(p, cfg, deltas) {
			continue
		}
		for s, needQty := range deltas[p].Needs {
			if needQty > demand[s] {
				demand[s] = needQty
			}
		}
	}
	return demand
}

// ! calculates how much generator output should be stockpiled
// ! before a process consumes a root catalyst
func computeCatalystThreshold(cfg *models.Config, deltas []ProcessDeltas) ([]bool, []bool, int) {
	numStocks := len(cfg.StockNames)
	isCat, isRootCat := make([]bool, numStocks), make([]bool, numStocks)
	maxBatch := 0
	for p := range cfg.Processes {
		for s, needQty := range deltas[p].Needs {
			if needQty > maxBatch {
				maxBatch = needQty
			}
			if needQty > 0 && deltas[p].Net[s] == 0 {
				isCat[s] = true
				if cfg.InitialStocks[s] > 0 {
					isRootCat[s] = true
				}
			}
		}
	}
	return isCat, isRootCat, maxBatch + 5
}

// * reports whether a non-root catalyst is still needed by an active converter
func isCatStillNeeded(s, needQty int, cfg *models.Config, deltas []ProcessDeltas, stocks, inFlight []int) bool {
	for pUse := range cfg.Processes {
		if deltas[pUse].Needs[s] >= needQty && deltas[pUse].Net[s] == 0 {
			canStillUse := true
			for inS, inQty := range deltas[pUse].Needs {
				if inS != s && stocks[inS]+inFlight[inS] < inQty {
					canStillUse = false
				}
			}
			if canStillUse {
				return true
			}
		}
	}
	return false
}

// * reports whether consuming root catalyst s would starve raw generator output
func isRootCatPremature(s, needQty int, cfg *models.Config, deltas []ProcessDeltas, stocks, inFlight []int, catReserve int) bool {
	if stocks[s] > needQty {
		return false
	}
	for pGen := range cfg.Processes {
		if deltas[pGen].Needs[s] > 0 && deltas[pGen].Net[s] == 0 {
			for outS, netOut := range deltas[pGen].Net {
				if netOut > 0 && stocks[outS]+inFlight[outS] < catReserve {
					return true
				}
			}
		}
	}
	return false
}

// ! prevents consuming a root catalyst before enough raw stock is accumulated
func isBlockedByCatalystGuard(p int, cfg *models.Config, deltas []ProcessDeltas, stocks, inFlight, demand []int, isCat, isRootCat []bool, catReserve int) bool {
	if hasFastCatalystProducer(p, cfg, deltas) {
		for c := range cfg.StockNames {
			if isCat[c] && !isRootCat[c] && stocks[c]+inFlight[c] > 0 {
				return true
			}
		}
	}
	d := &deltas[p]
	for s, needQty := range d.Needs {
		if needQty > 0 && d.Net[s] < 0 && isCat[s] && !isRootCat[s] {
			if isCatStillNeeded(s, needQty, cfg, deltas, stocks, inFlight) {
				return true
			}
		}
		if needQty > 0 && d.Net[s] < 0 && isRootCat[s] {
			if isRootCatPremature(s, needQty, cfg, deltas, stocks, inFlight, catReserve) {
				return true
			}
		}
		if needQty > 0 && d.Net[s] == 0 && stocks[s]+inFlight[s] < demand[s] && needQty < demand[s] {
			for pUp := range cfg.Processes {
				if deltas[pUp].Net[s] > 0 && deltas[pUp].Needs[s] > 0 {
					for inS, inQty := range deltas[pUp].Needs {
						if inS != s && inQty > 0 && (d.Net[inS] <= 0 || stocks[inS]+inFlight[inS] >= inQty) {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// determines if a process is producing intermediate stocks whose downstream demand is already met
func isProcessCapped(p int, cfg *models.Config, deltas []ProcessDeltas, stocks, inFlight, demand []int, isTarget, isRootCat []bool, catReserve int) bool {
	if len(cfg.Targets) == 0 {
		return false
	}
	d := &deltas[p]
	for s, needQty := range d.Needs {
		if needQty > 0 && d.Net[s] == 0 && isRootCat[s] {
			for outS, netQty := range d.Net {
				if netQty > 0 && stocks[outS]+inFlight[outS] < catReserve {
					return false
				}
			}
		}
	}
	hasIntermediate, allSatisfied := false, true
	for s, netQty := range d.Net {
		if netQty <= 0 {
			continue
		}
		if s >= 0 && s < len(isTarget) && isTarget[s] {
			return false
		}
		if dem := demand[s]; dem > 0 {
			hasIntermediate = true
			if stocks[s]+inFlight[s] < dem {
				allSatisfied = false
			}
		}
	}
	return hasIntermediate && allSatisfied
}
