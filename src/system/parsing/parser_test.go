package parsing

import (
	"strings"
	"testing"

	"krpsim/src/system/models"
)

// TestParseAllResourceFiles verifies that all 6 sample resource files parse correctly.
func TestParseAllResourceFiles(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedProc int
		optimizeTime bool
		targetNames  []string
	}{
		{"simple", "../../../resources/simple", 3, true, []string{"client_content"}},
		{"ikea", "../../../resources/ikea", 4, true, []string{"armoire"}},
		{"recre", "../../../resources/recre", 5, false, []string{"marelle"}},
		{"pomme", "../../../resources/pomme", 18, false, []string{"euro"}},
		{"steak", "../../../resources/steak", 5, true, []string{"steak_cuit"}},
		{"inception", "../../../resources/inception", 11, false, []string{"year"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := ParseFile(tc.path)
			if err != nil {
				t.Fatalf("ParseFile(%q) unexpected error: %v", tc.path, err)
			}
			if len(cfg.Processes) != tc.expectedProc {
				t.Errorf("expected %d processes, got %d", tc.expectedProc, len(cfg.Processes))
			}
			if cfg.OptimizeTime != tc.optimizeTime {
				t.Errorf("expected OptimizeTime=%v, got %v", tc.optimizeTime, cfg.OptimizeTime)
			}
			assertTargets(t, cfg, tc.targetNames)
		})
	}
}

func assertTargets(t *testing.T, cfg *models.Config, expectedNames []string) {
	t.Helper()
	if len(cfg.Targets) != len(expectedNames) {
		t.Fatalf("expected %d targets, got %d", len(expectedNames), len(cfg.Targets))
	}
	for i, name := range expectedNames {
		expectedID, exists := cfg.StockIndex[name]
		if !exists {
			t.Fatalf("expected target stock %q to exist in StockIndex", name)
		}
		if cfg.Targets[i] != expectedID {
			t.Errorf("target[%d] = %d, expected %d (%s)", i, cfg.Targets[i], expectedID, name)
		}
	}
}

// TestSimpleResourceDetails checks exact fields of the simple resource.
func TestSimpleResourceDetails(t *testing.T) {
	cfg, err := ParseFile("../../../resources/simple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InitialStocks[cfg.StockIndex["euro"]] != 10 {
		t.Errorf("euro initial stock = %d, want 10", cfg.InitialStocks[cfg.StockIndex["euro"]])
	}
	if cfg.InitialStocks[cfg.StockIndex["materiel"]] != 0 {
		t.Errorf("materiel initial stock = %d, want 0", cfg.InitialStocks[cfg.StockIndex["materiel"]])
	}

	proc0 := cfg.Processes[0]
	if proc0.Name != "achat_materiel" || proc0.Delay != 10 {
		t.Errorf("unexpected proc0: %+v", proc0)
	}
	if len(proc0.Needs) != 1 || proc0.Needs[0].Qty != 8 {
		t.Errorf("unexpected proc0 needs: %+v", proc0.Needs)
	}
	if len(proc0.Results) != 1 || proc0.Results[0].Qty != 1 {
		t.Errorf("unexpected proc0 results: %+v", proc0.Results)
	}
}

// TestRecreEmptyResults checks that empty results (e.g. `manger:(bonbon:1)::10`) parse as empty.
func TestRecreEmptyResults(t *testing.T) {
	cfg, err := ParseFile("../../../resources/recre")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mangerIdx, ok := cfg.ProcIndex["manger"]
	if !ok {
		t.Fatalf("proc 'manger' not found")
	}
	manger := cfg.Processes[mangerIdx]
	if len(manger.Results) != 0 {
		t.Errorf("expected manger to have 0 results, got %d", len(manger.Results))
	}
	if len(manger.Needs) != 1 || manger.Needs[0].Qty != 1 {
		t.Errorf("expected manger to have 1 bonbon need, got %+v", manger.Needs)
	}
}

// TestSplitProcess verifies parenthesis-aware splitting on ':'.
func TestSplitProcess(t *testing.T) {
	parts, err := SplitProcess("achat_materiel:(euro:8):(materiel:1):10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"achat_materiel", "(euro:8)", "(materiel:1)", "10"}
	for i, exp := range expected {
		if parts[i] != exp {
			t.Errorf("part[%d] = %q, want %q", i, parts[i], exp)
		}
	}

	emptyParts, err := SplitProcess("manger:(bonbon:1)::10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emptyParts[2] != "" {
		t.Errorf("expected empty results part, got %q", emptyParts[2])
	}
}

// TestSplitProcessErrors verifies error handling for malformed process lines.
func TestSplitProcessErrors(t *testing.T) {
	badInputs := []string{
		"unclosed:(need:1:(res:1):10",
		"unexpected:need:1):(res:1):10",
		"too_few:(need:1):10",
		"too_many:(need:1):(res:1):10:extra",
	}
	for _, in := range badInputs {
		if _, err := SplitProcess(in); err == nil {
			t.Errorf("SplitProcess(%q) expected error, got nil", in)
		}
	}
}

// TestStockDeltaMerging verifies that duplicate stocks in needs/results are merged and summed.
func TestStockDeltaMerging(t *testing.T) {
	input := `
stock1:5
p1:(a:2;b:3;a:4):(c:1;c:2):10
optimize:(time;c)
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proc := cfg.Processes[0]
	if len(proc.Needs) != 2 {
		t.Fatalf("expected 2 merged needs, got %d", len(proc.Needs))
	}
	aID := cfg.StockIndex["a"]
	bID := cfg.StockIndex["b"]
	cID := cfg.StockIndex["c"]

	if proc.Needs[0].Stock != aID || proc.Needs[0].Qty != 6 {
		t.Errorf("need[0] = %+v, want stock 'a' qty 6", proc.Needs[0])
	}
	if proc.Needs[1].Stock != bID || proc.Needs[1].Qty != 3 {
		t.Errorf("need[1] = %+v, want stock 'b' qty 3", proc.Needs[1])
	}
	if len(proc.Results) != 1 || proc.Results[0].Stock != cID || proc.Results[0].Qty != 3 {
		t.Errorf("results = %+v, want stock 'c' qty 3", proc.Results)
	}
}

// TestEmptyNeedsAndResultsVariants verifies empty and paren-enclosed empty needs and results.
func TestEmptyNeedsAndResultsVariants(t *testing.T) {
	input := `
p1:():():5
p2:::10
p3:( ):(  ):15
optimize:(time)
`
	cfg, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Processes) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(cfg.Processes))
	}
	for i, p := range cfg.Processes {
		if len(p.Needs) != 0 || len(p.Results) != 0 {
			t.Errorf("process %d (%s) expected empty needs/results, got needs=%+v results=%+v", i, p.Name, p.Needs, p.Results)
		}
	}
}

// TestDuplicateStockError verifies the required "line N: duplicate stock <name>" error.
func TestDuplicateStockError(t *testing.T) {
	input := `# Header
euro:10
bonbon:5
euro:20
p1:(bonbon:1):(euro:1):10
optimize:(time)
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatalf("expected error for duplicate stock, got nil")
	}
	expected := "line 4: duplicate stock euro"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// TestDuplicateProcessError verifies error on duplicate process names.
func TestDuplicateProcessError(t *testing.T) {
	input := `
s:10
p1:(s:1):(s:2):5
p1:(s:2):(s:3):10
optimize:(time)
`
	_, err := Parse(strings.NewReader(input))
	if err == nil {
		t.Fatalf("expected error for duplicate process, got nil")
	}
	expected := "duplicate process p1"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("expected error containing %q, got %q", expected, err.Error())
	}
}

// TestOptimizeDirectiveRules verifies single appearance and target deduplication.
func TestOptimizeDirectiveRules(t *testing.T) {
	// Deduplication of targets and multiple time mentions
	inputValid := `
p:(a:1):(b:1):10
optimize:(time;b;a;b;time)
`
	cfg, err := Parse(strings.NewReader(inputValid))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.OptimizeTime {
		t.Errorf("expected OptimizeTime = true")
	}
	if len(cfg.Targets) != 2 {
		t.Errorf("expected 2 unique targets, got %d", len(cfg.Targets))
	}

	// Duplicate optimize line
	inputDup := `
p:(a:1):(b:1):10
optimize:(time)
optimize:(a)
`
	if _, err := Parse(strings.NewReader(inputDup)); err == nil {
		t.Errorf("expected error for duplicate optimize directive, got nil")
	}

	// Missing optimize line
	inputMissing := `
p:(a:1):(b:1):10
`
	if _, err := Parse(strings.NewReader(inputMissing)); err == nil {
		t.Errorf("expected error for missing optimize directive, got nil")
	}
}

// TestZeroProcessesError verifies error when no processes exist.
func TestZeroProcessesError(t *testing.T) {
	input := `
euro:100
optimize:(time)
`
	if _, err := Parse(strings.NewReader(input)); err == nil {
		t.Errorf("expected error for 0 processes, got nil")
	}
}

// TestInvalidSyntaxEdgeCases verifies various syntax validation errors.
func TestInvalidSyntaxEdgeCases(t *testing.T) {
	badSnippets := []struct {
		desc  string
		input string
	}{
		{"negative initial stock", "s:-5\np:(s:1):(s:2):1\noptimize:(time)"},
		{"non-numeric stock", "s:abc\np:(s:1):(s:2):1\noptimize:(time)"},
		{"empty stock name", ":10\np:(s:1):(s:2):1\noptimize:(time)"},
		{"negative delay", "p:(s:1):(s:2):-10\noptimize:(time)"},
		{"zero quantity in needs", "p:(s:0):(s:2):10\noptimize:(time)"},
		{"negative quantity in needs", "p:(s:-2):(s:2):10\noptimize:(time)"},
		{"zero quantity in results", "p:(s:1):(s:0):10\noptimize:(time)"},
		{"invalid delay text", "p:(s:1):(s:2):ten\noptimize:(time)"},
		{"empty optimize directive", "p:(s:1):(s:2):10\noptimize:()"},
		{"optimize without parens", "p:(s:1):(s:2):10\noptimize:time"},
		{"empty item in optimize", "p:(s:1):(s:2):10\noptimize:(time;)"},
		{"single field without colon", "only_word\np:(s:1):(s:2):10\noptimize:(time)"},
		{"three fields without parens", "a:b:c\np:(s:1):(s:2):10\noptimize:(time)"},
	}

	for _, tc := range badSnippets {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(tc.input)); err == nil {
				t.Errorf("[%s] expected error, got nil", tc.desc)
			}
		})
	}
}

// TestParseFileNotFound verifies error returned when file does not exist.
func TestParseFileNotFound(t *testing.T) {
	if _, err := ParseFile("non_existent_file_path_12345"); err == nil {
		t.Errorf("expected error opening non-existent file, got nil")
	}
}
