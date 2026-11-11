package sim

import (
	"krpsim/internal/config"
	"testing"
)

func helperMakeConfig() *config.Config {
	return &config.Config{
		StockNames:    []string{"poele", "pate", "crepe", "farine", "lait"},
		StockIndex:    map[string]int{"poele": 0, "pate": 1, "crepe": 2, "farine": 3, "lait": 4},
		InitialStocks: []int{1, 2, 0, 10, 5},
		Processes: []config.Process{
			{
				Name:    "cuire_crepe",
				Needs:   []config.StockDelta{{Stock: 0, Qty: 1}, {Stock: 1, Qty: 1}},
				Results: []config.StockDelta{{Stock: 0, Qty: 1}, {Stock: 2, Qty: 1}},
				Delay:   5,
			},
			{
				Name:    "faire_pate",
				Needs:   []config.StockDelta{{Stock: 3, Qty: 2}, {Stock: 4, Qty: 1}},
				Results: []config.StockDelta{{Stock: 1, Qty: 2}},
				Delay:   2,
			},
			{
				Name:    "instant_mix",
				Needs:   []config.StockDelta{{Stock: 3, Qty: 1}},
				Results: []config.StockDelta{{Stock: 1, Qty: 1}},
				Delay:   0,
			},
			{
				Name:    "free_gift",
				Needs:   []config.StockDelta{},
				Results: []config.StockDelta{{Stock: 3, Qty: 5}},
				Delay:   1,
			},
		},
		ProcIndex: map[string]int{
			"cuire_crepe": 0,
			"faire_pate":  1,
			"instant_mix": 2,
			"free_gift":   3,
		},
	}
}

func TestNewState(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)
	if s == nil {
		t.Fatal("expected non-nil State")
	}
	if s.Cycle != 0 {
		t.Errorf("expected Cycle == 0, got %d", s.Cycle)
	}
	if len(s.Stocks) != len(cfg.InitialStocks) {
		t.Fatalf("expected %d stocks, got %d", len(cfg.InitialStocks), len(s.Stocks))
	}
	for i, v := range cfg.InitialStocks {
		if s.Stocks[i] != v {
			t.Errorf("stock %d mismatch: got %d, expected %d", i, s.Stocks[i], v)
		}
	}
	s.Stocks[0] = 999
	if cfg.InitialStocks[0] == 999 {
		t.Error("modifying State.Stocks mutated Config.InitialStocks (not cloned)")
	}
	if s.InFlightCount() != 0 {
		t.Errorf("expected 0 in flight, got %d", s.InFlightCount())
	}
	if next, ok := s.NextEvent(); ok || next != 0 {
		t.Errorf("expected NextEvent (0, false), got (%d, %t)", next, ok)
	}
	if nilState := NewState(nil); nilState != nil {
		t.Error("expected nil for NewState(nil)")
	}
}

func TestMaxRuns(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// cuire_crepe needs poele:1, pate:1. Stocks: poele=1, pate=2 -> max 1 run
	if runs := s.MaxRuns(0, nil); runs != 1 {
		t.Errorf("expected MaxRuns=1 for cuire_crepe, got %d", runs)
	}

	// faire_pate needs farine:2, lait:1. Stocks: farine=10, lait=5 -> max 5 runs
	if runs := s.MaxRuns(1, nil); runs != 5 {
		t.Errorf("expected MaxRuns=5 for faire_pate, got %d", runs)
	}

	// Custom stocks passed explicitly
	customStocks := []int{3, 10, 0, 0, 0}
	if runs := s.MaxRuns(0, customStocks); runs != 3 {
		t.Errorf("expected MaxRuns=3 with custom stocks, got %d", runs)
	}

	// free_gift has no needs -> DefaultMaxRunsCap
	if runs := s.MaxRuns(3, nil); runs != DefaultMaxRunsCap {
		t.Errorf("expected MaxRuns=%d for free_gift, got %d", DefaultMaxRunsCap, runs)
	}

	// Invalid process index or nil handling
	if runs := s.MaxRuns(-1, nil); runs != 0 {
		t.Errorf("expected 0 for invalid index, got %d", runs)
	}
	if runs := s.MaxRuns(99, nil); runs != 0 {
		t.Errorf("expected 0 for out of bound index, got %d", runs)
	}
	var nilState *State
	if runs := nilState.MaxRuns(0, nil); runs != 0 {
		t.Errorf("expected 0 for nil state, got %d", runs)
	}
}

func TestMaxRunsDuplicateNeeds(t *testing.T) {
	cfg := &config.Config{
		StockNames:    []string{"bois"},
		InitialStocks: []int{9},
		Processes: []config.Process{
			{
				Name:  "double_need",
				Needs: []config.StockDelta{{Stock: 0, Qty: 2}, {Stock: 0, Qty: 3}},
			},
		},
	}
	s := NewState(cfg)
	// total need is 5. Stocks is 9. Max runs should be 1 (9 / 5 = 1).
	if runs := s.MaxRuns(0, nil); runs != 1 {
		t.Errorf("expected MaxRuns=1 for combined need of 5 with 9 stock, got %d", runs)
	}
}

func TestLaunchValidationAndErrors(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// Insufficient stock: farine has 10, need 20
	cfg.Processes[1].Needs[0].Qty = 20
	err := s.Launch(1)
	if err == nil {
		t.Fatal("expected error for insufficient stock, got nil")
	}
	// Verify stocks untouched
	if s.Stocks[3] != 10 || s.Stocks[4] != 5 {
		t.Errorf("stocks were modified on failed launch: farine=%d, lait=%d", s.Stocks[3], s.Stocks[4])
	}

	// Invalid process index
	if err := s.Launch(-1); err == nil {
		t.Error("expected error for process index -1")
	}
	if err := s.Launch(50); err == nil {
		t.Error("expected error for out of bounds process index")
	}

	var nilState *State
	if err := nilState.Launch(0); err == nil {
		t.Error("expected error for nil state")
	}
}

func TestLaunchZeroDelay(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// instant_mix: needs farine:1 (index 3), results pate:1 (index 1), Delay: 0
	initFarine := s.Stocks[3]
	initPate := s.Stocks[1]

	if err := s.Launch(2); err != nil {
		t.Fatalf("Launch instant_mix failed: %v", err)
	}

	if s.Stocks[3] != initFarine-1 {
		t.Errorf("expected farine to decrease by 1, got %d", s.Stocks[3])
	}
	if s.Stocks[1] != initPate+1 {
		t.Errorf("expected pate to increase immediately by 1, got %d", s.Stocks[1])
	}
	if s.InFlightCount() != 0 {
		t.Errorf("expected 0 in flight for 0-delay process, got %d", s.InFlightCount())
	}
	if s.Cycle != 0 {
		t.Errorf("expected cycle to remain 0, got %d", s.Cycle)
	}
}

func TestCatalystLocking(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// Stock: poele=1, pate=2, crepe=0. cuire_crepe needs poele:1, pate:1, results poele:1, crepe:1, delay=5
	if err := s.Launch(0); err != nil {
		t.Fatalf("first launch failed: %v", err)
	}
	if s.Stocks[0] != 0 {
		t.Errorf("catalyst poele should be locked (0), got %d", s.Stocks[0])
	}
	if s.Stocks[1] != 1 {
		t.Errorf("pate should be 1, got %d", s.Stocks[1])
	}
	if s.InFlightCount() != 1 {
		t.Errorf("expected 1 in flight, got %d", s.InFlightCount())
	}

	// Attempting second launch with poele locked must fail
	if err := s.Launch(0); err == nil {
		t.Fatal("expected failure launching cuire_crepe while catalyst is locked")
	}

	// Advance to cycle 4: event still pending
	s.AdvanceTo(4)
	if s.Cycle != 4 {
		t.Errorf("expected cycle 4, got %d", s.Cycle)
	}
	if s.Stocks[0] != 0 {
		t.Errorf("poele must still be 0 at cycle 4, got %d", s.Stocks[0])
	}
	if err := s.Launch(0); err == nil {
		t.Fatal("second launch should still fail at cycle 4")
	}

	// Advance to cycle 5: catalyst returned and crepe credited
	s.AdvanceTo(5)
	if s.Stocks[0] != 1 {
		t.Errorf("poele should be unlocked (1) at cycle 5, got %d", s.Stocks[0])
	}
	if s.Stocks[2] != 1 {
		t.Errorf("crepe should be 1 at cycle 5, got %d", s.Stocks[2])
	}
	if s.InFlightCount() != 0 {
		t.Errorf("expected 0 in flight at cycle 5, got %d", s.InFlightCount())
	}

	// Now second launch can proceed
	if err := s.Launch(0); err != nil {
		t.Fatalf("second launch at cycle 5 failed: %v", err)
	}
	if s.Stocks[0] != 0 {
		t.Errorf("poele locked again (0), got %d", s.Stocks[0])
	}

	// Drain remaining events
	endCycle := s.Drain()
	if endCycle != 10 {
		t.Errorf("expected Drain to complete at cycle 10, got %d", endCycle)
	}
	if s.Stocks[0] != 1 || s.Stocks[2] != 2 {
		t.Errorf("final stocks unexpected: poele=%d, crepe=%d", s.Stocks[0], s.Stocks[2])
	}
}

func TestAdvanceToMultipleEvents(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// Launch faire_pate (delay 2), then free_gift (delay 1)
	if err := s.Launch(1); err != nil {
		t.Fatalf("launch faire_pate failed: %v", err)
	}
	if err := s.Launch(3); err != nil {
		t.Fatalf("launch free_gift failed: %v", err)
	}

	if next, ok := s.NextEvent(); !ok || next != 1 {
		t.Errorf("expected next event at cycle 1, got (%d, %t)", next, ok)
	}

	// Advance to 1: free_gift finishes (end 1)
	s.AdvanceTo(1)
	if s.InFlightCount() != 1 {
		t.Errorf("expected 1 in flight at cycle 1, got %d", s.InFlightCount())
	}
	if s.Stocks[3] != 10-2+5 { // 10 - 2 (pate) + 5 (gift)
		t.Errorf("expected farine=13, got %d", s.Stocks[3])
	}

	// Advance to 2: faire_pate finishes (end 2)
	s.AdvanceTo(2)
	if s.InFlightCount() != 0 {
		t.Errorf("expected 0 in flight at cycle 2, got %d", s.InFlightCount())
	}
	if s.Stocks[1] != 2+2 { // initial 2 + 2 created
		t.Errorf("expected pate=4, got %d", s.Stocks[1])
	}

	// Advance to cycle 10 with no events: simply updates cycle
	s.AdvanceTo(10)
	if s.Cycle != 10 {
		t.Errorf("expected cycle 10, got %d", s.Cycle)
	}
}

func TestDrain(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// Calling drain on empty state
	s.Cycle = 7
	if final := s.Drain(); final != 7 {
		t.Errorf("expected Drain on empty state to return 7, got %d", final)
	}

	// Reset cycle to 0 and queue multiple events
	s.Cycle = 0
	_ = s.Launch(1) // delay 2 -> End: 2
	_ = s.Launch(3) // delay 1 -> End: 1
	_ = s.Launch(0) // delay 5 -> End: 5

	if s.InFlightCount() != 3 {
		t.Fatalf("expected 3 in flight, got %d", s.InFlightCount())
	}

	end := s.Drain()
	if end != 5 {
		t.Errorf("expected Drain to return 5, got %d", end)
	}
	if s.Cycle != 5 {
		t.Errorf("expected s.Cycle=5 after Drain, got %d", s.Cycle)
	}
	if s.InFlightCount() != 0 {
		t.Errorf("expected 0 in flight after Drain, got %d", s.InFlightCount())
	}
}

func TestHeapOrdering(t *testing.T) {
	var h eventHeap
	h.push(Event{End: 10, Proc: 2})
	h.push(Event{End: 3, Proc: 5})
	h.push(Event{End: 7, Proc: 1})
	h.push(Event{End: 3, Proc: 1})

	// Pop 1: End 3, Proc 1
	e1 := h.pop()
	if e1.End != 3 || e1.Proc != 1 {
		t.Errorf("expected (3, 1), got (%d, %d)", e1.End, e1.Proc)
	}
	// Pop 2: End 3, Proc 5
	e2 := h.pop()
	if e2.End != 3 || e2.Proc != 5 {
		t.Errorf("expected (3, 5), got (%d, %d)", e2.End, e2.Proc)
	}
	// Pop 3: End 7, Proc 1
	e3 := h.pop()
	if e3.End != 7 || e3.Proc != 1 {
		t.Errorf("expected (7, 1), got (%d, %d)", e3.End, e3.Proc)
	}
	// Pop 4: End 10, Proc 2
	e4 := h.pop()
	if e4.End != 10 || e4.Proc != 2 {
		t.Errorf("expected (10, 2), got (%d, %d)", e4.End, e4.Proc)
	}
	// Pop empty heap
	empty := h.pop()
	if empty.End != 0 || empty.Proc != 0 {
		t.Errorf("expected zero Event from empty pop, got %+v", empty)
	}

	// Peek empty heap
	if _, ok := h.peek(); ok {
		t.Error("expected ok=false for peek on empty heap")
	}
}

func TestNilSafety(t *testing.T) {
	var s *State
	s.AdvanceTo(10)
	if next, ok := s.NextEvent(); ok || next != 0 {
		t.Errorf("expected (0, false), got (%d, %t)", next, ok)
	}
	if drained := s.Drain(); drained != 0 {
		t.Errorf("expected 0, got %d", drained)
	}
	if count := s.InFlightCount(); count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
	s.creditResults(nil)
}

func TestCreditResultsExtendsStocks(t *testing.T) {
	cfg := &config.Config{
		StockNames:    []string{"a"},
		InitialStocks: []int{5},
		Processes: []config.Process{
			{
				Name:    "produce_new_stock",
				Needs:   []config.StockDelta{},
				Results: []config.StockDelta{{Stock: 3, Qty: 10}}, // Stock index 3 > len(Stocks)=1
				Delay:   0,
			},
		},
	}
	s := NewState(cfg)
	if err := s.Launch(0); err != nil {
		t.Fatalf("launch failed: %v", err)
	}
	if len(s.Stocks) < 4 {
		t.Fatalf("expected Stocks to be expanded to at least 4, got %d", len(s.Stocks))
	}
	if s.Stocks[3] != 10 {
		t.Errorf("expected Stocks[3] = 10, got %d", s.Stocks[3])
	}
}

func TestInFlightStocks(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)
	if err := s.Launch(0); err != nil { // cuire_crepe: Delay 5, Results poele:1, crepe:1
		t.Fatalf("launch cuire_crepe failed: %v", err)
	}
	inFlight := make([]int, len(cfg.StockNames))
	s.InFlightStocks(inFlight)
	if inFlight[0] != 1 || inFlight[2] != 1 {
		t.Errorf("expected inFlight poele=1, crepe=1, got poele=%d, crepe=%d", inFlight[0], inFlight[2])
	}
	s.AdvanceTo(5)
	s.InFlightStocks(inFlight)
	if inFlight[0] != 0 || inFlight[2] != 0 {
		t.Errorf("expected inFlight cleared after AdvanceTo, got poele=%d, crepe=%d", inFlight[0], inFlight[2])
	}
}
