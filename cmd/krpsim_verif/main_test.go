package main

import (
	"strings"
	"testing"

	"krpsim/src/system/parsing"
)

func loadTestConfig(t *testing.T, paths ...string) string {
	t.Helper()
	for _, p := range paths {
		if _, err := parsing.ParseFile(p); err == nil {
			return p
		}
	}
	t.Fatalf("could not load config from any of %v", paths)
	return ""
}

func TestVerifyTrace_SimpleValidAndBanners(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	trace := `
> 【=◈︿◈=】= 3 processes, 3 stocks, 2 to optimize
Evaluating .................. done.
【=◈︿◈=】
0:achat_materiel
10:realisation_produit # inline comment should be ignored
40:livraison
no more process doable at time 61
Stock :
 euro => 2
 materiel => 0
 produit => 0
 client => 1
`
	res, err := verifyTrace(cfg, strings.NewReader(trace))
	if err != nil {
		t.Fatalf("verifyTrace returned unexpected I/O error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected valid trace, got error at cycle %d (%s): %s", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
	if res.LastCycle != 60 {
		t.Errorf("expected LastCycle=60, got %d", res.LastCycle)
	}
	if got := res.Stocks[cfg.StockIndex["euro"]]; got != 2 {
		t.Errorf("expected euro=2, got %d", got)
	}
	if got := res.Stocks[cfg.StockIndex["client_content"]]; got != 1 {
		t.Errorf("expected client_content=1, got %d", got)
	}
}

func TestVerifyTrace_InsufficientStock(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// achat_materiel takes 10 cycles, so at cycle 5 materiel is still 0
	trace := "0:achat_materiel\n5:realisation_produit\n"
	res, err := verifyTrace(cfg, strings.NewReader(trace))
	if err != nil {
		t.Fatalf("verifyTrace returned unexpected I/O error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected trace verification to fail due to insufficient stock at cycle 5")
	}
	if res.ErrCycle != 5 || res.ErrProc != "realisation_produit" {
		t.Errorf("expected error at (5, realisation_produit), got (%d, %q): %s", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
}

func TestVerifyTrace_OutOfOrderCycle(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	trace := "10:achat_materiel\n5:achat_materiel\n"
	res, err := verifyTrace(cfg, strings.NewReader(trace))
	if err != nil {
		t.Fatalf("verifyTrace returned unexpected I/O error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected trace verification to fail on out-of-order cycle")
	}
	if res.ErrCycle != 5 || res.ErrProc != "achat_materiel" {
		t.Errorf("expected error at (5, achat_materiel), got (%d, %q): %s", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
}

func TestVerifyTrace_UnknownProcess(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	trace := "0:non_existent_proc\n"
	res, err := verifyTrace(cfg, strings.NewReader(trace))
	if err != nil {
		t.Fatalf("verifyTrace returned unexpected I/O error: %v", err)
	}
	if res.OK {
		t.Fatalf("expected trace verification to fail on unknown process")
	}
	if res.ErrCycle != 0 || res.ErrProc != "non_existent_proc" {
		t.Errorf("expected error at (0, non_existent_proc), got (%d, %q): %s", res.ErrCycle, res.ErrProc, res.ErrReason)
	}
}

func TestVerifyTrace_MalformedLines(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	cases := []string{
		"-1:achat_materiel\n",
		"abc:achat_materiel\n",
		"10:\n",
		"random_garbage_without_colon\n",
	}
	for _, tc := range cases {
		res, err := verifyTrace(cfg, strings.NewReader(tc))
		if err != nil {
			t.Fatalf("unexpected I/O error for %q: %v", tc, err)
		}
		if res.OK {
			t.Errorf("expected malformed trace %q to fail verification, but got OK", tc)
		}
	}
}

func TestVerifyTrace_ZeroDelayProcess(t *testing.T) {
	cfgText := `
seed:1
step1:(seed:1):(mid:1):0
step2:(mid:1):(goal:1):5
optimize:(goal)
`
	cfg, err := parsing.Parse(strings.NewReader(cfgText))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// step1 has delay 0, so mid:1 is immediately available at cycle 0 for step2
	trace := "0:step1\n0:step2\n"
	res, err := verifyTrace(cfg, strings.NewReader(trace))
	if err != nil {
		t.Fatalf("verifyTrace error: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected valid trace for 0-delay chaining, got: %s", res.ErrReason)
	}
	if res.LastCycle != 5 {
		t.Errorf("expected LastCycle=5, got %d", res.LastCycle)
	}
	if got := res.Stocks[cfg.StockIndex["goal"]]; got != 1 {
		t.Errorf("expected goal=1, got %d", got)
	}
}

func TestPrintVerifOutput(t *testing.T) {
	path := loadTestConfig(t, "../../resources/simple", "resources/simple")
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	var buf strings.Builder
	printVerifOutput(&buf, cfg, VerifyResult{
		OK:        false,
		ErrCycle:  5,
		ErrProc:   "realisation_produit",
		ErrReason: "insufficient stock",
		LastCycle: 5,
		Stocks:    []int{2, 0, 0, 0},
	})
	out := buf.String()
	if !strings.Contains(out, "Trace error at cycle 5") || !strings.Contains(out, "Last cycle: 5") || !strings.Contains(out, "euro => 2") {
		t.Errorf("unexpected printVerifOutput output:\n%s", out)
	}
}
