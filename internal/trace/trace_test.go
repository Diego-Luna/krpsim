package trace

import "testing"

func TestOwnBannerIsSkipped(t *testing.T) {
	lines := []string{
		"> " + Mascot + "= 3 processes, 4 stocks, 2 to optimize",
		Evaluating,
		Mascot,
		Finished + " 61",
		StockHeader,
		" euro " + StockArrow + " 2",
	}
	for _, l := range lines {
		if _, skip, err := Parse(l); !skip || err != nil {
			t.Errorf("banner line %q not skipped (err=%v)", l, err)
		}
	}
}

func TestStepRoundTrip(t *testing.T) {
	in := Step{Cycle: 42, Proc: "do_montant"}
	out, skip, err := Parse(in.String())
	if err != nil || skip || out != in {
		t.Fatalf("got %+v skip=%v err=%v, want %+v", out, skip, err, in)
	}
}

func TestParseRejects(t *testing.T) {
	for _, l := range []string{"-1:a", "abc:a", "10:", "garbage"} {
		if _, skip, err := Parse(l); skip || err == nil {
			t.Errorf("expected error for %q", l)
		}
	}
}
