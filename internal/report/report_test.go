package report

import (
	"strings"
	"testing"

	"krpsim/internal/config"
	"krpsim/internal/verify"
)

func TestVerifyOutput(t *testing.T) {
	cfg, err := config.ParseFile("../../resources/simple")
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	var buf strings.Builder
	Verify(&buf, cfg, verify.Result{
		ErrCycle:  5,
		ErrProc:   "realisation_produit",
		ErrReason: "insufficient stock",
		LastCycle: 5,
		Stocks:    []int{2, 0, 0, 0},
	})
	out := buf.String()
	for _, want := range []string{"Trace error at cycle 5", "Last cycle: 5", "euro => 2"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}
}
