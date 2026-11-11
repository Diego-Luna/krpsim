package sim

import (
	"errors"
	"fmt"
	"krpsim/internal/config"
)

// * is returned by `MaxRuns“ when a process has no needs
const DefaultMaxRunsCap = 10000

// * State represents the simulation state at a specific cycle
type State struct {
	Cfg    *config.Config
	Cycle  int
	Stocks []int
	events eventHeap
}

// ! NewState initializes a simulation state from a configuration, cloning initial stocks
func NewState(cfg *config.Config) *State {
	if cfg == nil {
		return nil
	}
	stocks := make([]int, len(cfg.InitialStocks))
	copy(stocks, cfg.InitialStocks)
	return &State{
		Cfg:    cfg,
		Cycle:  0,
		Stocks: stocks,
		events: make(eventHeap, 0),
	}
}

// ! MaxRuns returns how many times process p can run given the provided stocks slice
// ! If stocks is nil, the state's current stocks are used
// ! If p has no needs, it returns DefaultMaxRunsCap (10000)
func (s *State) MaxRuns(p int, stocks []int) int {
	if s == nil || s.Cfg == nil || p < 0 || p >= len(s.Cfg.Processes) {
		return 0
	}
	if stocks == nil {
		stocks = s.Stocks
	}
	proc := &s.Cfg.Processes[p]
	if len(proc.Needs) == 0 {
		return DefaultMaxRunsCap
	}

	maxRuns := DefaultMaxRunsCap
	hasPositiveNeed := false
	for i := 0; i < len(proc.Needs); i++ {
		stockIdx, totalNeed := s.aggregatedNeed(proc, i)
		if totalNeed <= 0 {
			continue
		}
		hasPositiveNeed = true
		if stockIdx < 0 || stockIdx >= len(stocks) {
			return 0
		}
		available := stocks[stockIdx]
		if available <= 0 {
			return 0
		}
		if runs := available / totalNeed; runs < maxRuns {
			maxRuns = runs
		}
	}
	if !hasPositiveNeed {
		return DefaultMaxRunsCap
	}
	return maxRuns
}

// Launch checks if Stocks has enough for proc.Needs, deducts them, and either credits
// results immediately (if Delay == 0) or schedules a completion event in the heap
func (s *State) Launch(p int) error {
	if s == nil || s.Cfg == nil {
		return errors.New("nil state or config")
	}
	if p < 0 || p >= len(s.Cfg.Processes) {
		return fmt.Errorf("process index %d out of bounds [0, %d)", p, len(s.Cfg.Processes))
	}
	proc := &s.Cfg.Processes[p]
	if err := s.checkNeeds(proc); err != nil {
		return err
	}
	for _, need := range proc.Needs {
		if need.Qty > 0 && need.Stock >= 0 && need.Stock < len(s.Stocks) {
			s.Stocks[need.Stock] -= need.Qty
		}
	}
	if proc.Delay <= 0 {
		s.creditResults(proc)
	} else {
		s.events.push(Event{
			End:  s.Cycle + proc.Delay,
			Proc: p,
		})
	}
	return nil
}

// * AdvanceTo pops and credits results for all events with End <= cycle, then sets Cycle = cycle.
func (s *State) AdvanceTo(cycle int) {
	if s == nil || s.Cfg == nil {
		return
	}
	for len(s.events) > 0 {
		ev, _ := s.events.peek()
		if ev.End > cycle {
			break
		}
		s.events.pop()
		if ev.Proc >= 0 && ev.Proc < len(s.Cfg.Processes) {
			s.creditResults(&s.Cfg.Processes[ev.Proc])
		}
	}
	s.Cycle = cycle
}

// * NextEvent! returns the earliest event completion cycle, or (0, false) if no events are pending
func (s *State) NextEvent() (int, bool) {
	if s == nil || len(s.events) == 0 {
		return 0, false
	}
	ev, ok := s.events.peek()
	if !ok {
		return 0, false
	}
	return ev.End, true
}

// * Drain pops and credits all remaining events in chronological order, updating Cycle to the last event's End, and returns the final Cycle
func (s *State) Drain() int {
	if s == nil || s.Cfg == nil {
		return 0
	}
	for len(s.events) > 0 {
		ev := s.events.pop()
		if ev.Proc >= 0 && ev.Proc < len(s.Cfg.Processes) {
			s.creditResults(&s.Cfg.Processes[ev.Proc])
		}
		s.Cycle = ev.End
	}
	return s.Cycle
}

// * InFlightCount returns the number of active, incomplete process events currently in the heap
func (s *State) InFlightCount() int {
	if s == nil {
		return 0
	}
	return len(s.events)
}
