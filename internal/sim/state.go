package sim

import (
	"fmt"

	"krpsim/internal/config"
)

// State is the stock situation at Cycle plus every process still running.
// Needs are taken when a process starts, results arrive at start + delay.
type State struct {
	Cfg    *config.Config
	Cycle  int
	Stocks []int
	events eventHeap
}

func NewState(cfg *config.Config) *State {
	return &State{Cfg: cfg, Stocks: append([]int(nil), cfg.InitialStocks...)}
}

func (s *State) CanLaunch(p int) bool {
	for _, n := range s.Cfg.Processes[p].Needs {
		if s.Stocks[n.Stock] < n.Qty {
			return false
		}
	}
	return true
}

func (s *State) Launch(p int) error {
	proc := &s.Cfg.Processes[p]
	for _, n := range proc.Needs {
		if have := s.Stocks[n.Stock]; have < n.Qty {
			return fmt.Errorf("insufficient stock for %s: have %d, need %d", s.Cfg.StockNames[n.Stock], have, n.Qty)
		}
	}
	for _, n := range proc.Needs {
		s.Stocks[n.Stock] -= n.Qty
	}
	if proc.Delay == 0 {
		s.credit(proc)
		return nil
	}
	s.events.push(event{end: s.Cycle + proc.Delay, proc: p})
	return nil
}

// AdvanceTo moves the clock to cycle and books every result due by then.
func (s *State) AdvanceTo(cycle int) {
	for len(s.events) > 0 && s.events[0].end <= cycle {
		s.credit(&s.Cfg.Processes[s.events.pop().proc])
	}
	s.Cycle = cycle
}

func (s *State) NextEvent() (int, bool) {
	if len(s.events) == 0 {
		return 0, false
	}
	return s.events[0].end, true
}

// Drain lets every running process finish and returns the cycle the last one ends.
func (s *State) Drain() int {
	for len(s.events) > 0 {
		ev := s.events.pop()
		s.credit(&s.Cfg.Processes[ev.proc])
		s.Cycle = ev.end
	}
	return s.Cycle
}

func (s *State) credit(proc *config.Process) {
	for _, r := range proc.Results {
		s.Stocks[r.Stock] += r.Qty
	}
}
