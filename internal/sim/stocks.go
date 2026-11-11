package sim

import (
	"fmt"
	"krpsim/internal/config"
)

// i returns stock index and total quantity needed across proc.Needs
func (s *State) aggregatedNeed(proc *config.Process, idx int) (int, int) {
	stockIdx := proc.Needs[idx].Stock
	qty := proc.Needs[idx].Qty
	if qty <= 0 {
		return stockIdx, 0
	}
	for k := 0; k < idx; k++ {
		if proc.Needs[k].Stock == stockIdx {
			return stockIdx, 0
		}
	}
	totalNeed := qty
	for j := idx + 1; j < len(proc.Needs); j++ {
		if proc.Needs[j].Stock == stockIdx && proc.Needs[j].Qty > 0 {
			totalNeed += proc.Needs[j].Qty
		}
	}
	return stockIdx, totalNeed
}

// we verify that s.Stocks contains enough stock for proc.Needs
func (s *State) checkNeeds(proc *config.Process) error {
	for i := 0; i < len(proc.Needs); i++ {
		stockIdx, totalNeed := s.aggregatedNeed(proc, i)
		if totalNeed <= 0 {
			continue
		}
		have := 0
		if stockIdx >= 0 && stockIdx < len(s.Stocks) {
			have = s.Stocks[stockIdx]
		}
		if stockIdx < 0 || stockIdx >= len(s.Stocks) || have < totalNeed {
			name := fmt.Sprintf("#%d", stockIdx)
			if stockIdx >= 0 && stockIdx < len(s.Cfg.StockNames) {
				name = s.Cfg.StockNames[stockIdx]
			}
			return fmt.Errorf("insufficient stock for %s: have %d, need %d", name, have, totalNeed)
		}
	}
	return nil
}

// we adds process result quantities to state stocks
func (s *State) creditResults(proc *config.Process) {
	if proc == nil {
		return
	}
	for _, res := range proc.Results {
		if res.Stock < 0 || res.Qty <= 0 {
			continue
		}
		if res.Stock >= len(s.Stocks) {
			newStocks := make([]int, res.Stock+1)
			copy(newStocks, s.Stocks)
			s.Stocks = newStocks
		}
		s.Stocks[res.Stock] += res.Qty
	}
}

// ! InFlightStocks populates dst with total quantities currently in flight in s.events
func (s *State) InFlightStocks(dst []int) {
	for i := range dst {
		dst[i] = 0
	}
	if s == nil || s.Cfg == nil {
		return
	}
	for _, ev := range s.events {
		if ev.Proc >= 0 && ev.Proc < len(s.Cfg.Processes) {
			for _, res := range s.Cfg.Processes[ev.Proc].Results {
				if res.Stock >= 0 && res.Stock < len(dst) {
					dst[res.Stock] += res.Qty
				}
			}
		}
	}
}
