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
	if len(s.events) != 0 {
		t.Errorf("expected 0 in flight, got %d", len(s.events))
	}
	if next, ok := s.NextEvent(); ok || next != 0 {
		t.Errorf("expected NextEvent (0, false), got (%d, %t)", next, ok)
	}
}

func TestCanLaunch(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)

	// cuire_crepe needs poele:1, pate:1 and both are there
	if !s.CanLaunch(0) {
		t.Error("expected cuire_crepe to be launchable")
	}
	// free_gift has no needs at all
	if !s.CanLaunch(3) {
		t.Error("expected free_gift to be launchable")
	}

	s.Stocks[0] = 0
	if s.CanLaunch(0) {
		t.Error("expected cuire_crepe to be blocked without poele")
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
	if len(s.events) != 0 {
		t.Errorf("expected 0 in flight for 0-delay process, got %d", len(s.events))
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
	if len(s.events) != 1 {
		t.Errorf("expected 1 in flight, got %d", len(s.events))
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
	if len(s.events) != 0 {
		t.Errorf("expected 0 in flight at cycle 5, got %d", len(s.events))
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
	if len(s.events) != 1 {
		t.Errorf("expected 1 in flight at cycle 1, got %d", len(s.events))
	}
	if s.Stocks[3] != 10-2+5 { // 10 - 2 (pate) + 5 (gift)
		t.Errorf("expected farine=13, got %d", s.Stocks[3])
	}

	// Advance to 2: faire_pate finishes (end 2)
	s.AdvanceTo(2)
	if len(s.events) != 0 {
		t.Errorf("expected 0 in flight at cycle 2, got %d", len(s.events))
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

	if len(s.events) != 3 {
		t.Fatalf("expected 3 in flight, got %d", len(s.events))
	}

	end := s.Drain()
	if end != 5 {
		t.Errorf("expected Drain to return 5, got %d", end)
	}
	if s.Cycle != 5 {
		t.Errorf("expected s.Cycle=5 after Drain, got %d", s.Cycle)
	}
	if len(s.events) != 0 {
		t.Errorf("expected 0 in flight after Drain, got %d", len(s.events))
	}
}

func TestHeapOrdering(t *testing.T) {
	var h eventHeap
	h.push(event{end: 10, proc: 2})
	h.push(event{end: 3, proc: 5})
	h.push(event{end: 7, proc: 1})
	h.push(event{end: 3, proc: 1})

	// Pop 1: End 3, Proc 1
	e1 := h.pop()
	if e1.end != 3 || e1.proc != 1 {
		t.Errorf("expected (3, 1), got (%d, %d)", e1.end, e1.proc)
	}
	// Pop 2: End 3, Proc 5
	e2 := h.pop()
	if e2.end != 3 || e2.proc != 5 {
		t.Errorf("expected (3, 5), got (%d, %d)", e2.end, e2.proc)
	}
	// Pop 3: End 7, Proc 1
	e3 := h.pop()
	if e3.end != 7 || e3.proc != 1 {
		t.Errorf("expected (7, 1), got (%d, %d)", e3.end, e3.proc)
	}
	// Pop 4: End 10, Proc 2
	e4 := h.pop()
	if e4.end != 10 || e4.proc != 2 {
		t.Errorf("expected (10, 2), got (%d, %d)", e4.end, e4.proc)
	}
	// Pop empty heap
	empty := h.pop()
	if empty != (event{}) {
		t.Errorf("expected zero Event from empty pop, got %+v", empty)
	}
}

func TestInFlightStocks(t *testing.T) {
	cfg := helperMakeConfig()
	s := NewState(cfg)
	if err := s.Launch(0); err != nil { // cuire_crepe: Delay 5, Results poele:1, crepe:1
		t.Fatalf("launch cuire_crepe failed: %v", err)
	}
	inFlight := make([]int, len(cfg.StockNames))
	s.InFlight(inFlight)
	if inFlight[0] != 1 || inFlight[2] != 1 {
		t.Errorf("expected inFlight poele=1, crepe=1, got poele=%d, crepe=%d", inFlight[0], inFlight[2])
	}
	s.AdvanceTo(5)
	s.InFlight(inFlight)
	if inFlight[0] != 0 || inFlight[2] != 0 {
		t.Errorf("expected inFlight cleared after AdvanceTo, got poele=%d, crepe=%d", inFlight[0], inFlight[2])
	}
}
