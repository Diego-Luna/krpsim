package cmd

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"krpsim/src/system/engine"
	"krpsim/src/system/models"
	"krpsim/src/system/parsing"
)

func findConfigFile(tb testing.TB, relPaths ...string) string {
	tb.Helper()
	for _, p := range relPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	tb.Fatalf("could not find file in paths: %v", relPaths)
	return ""
}

func loadConfig(tb testing.TB, relPaths ...string) *models.Config {
	tb.Helper()
	path := findConfigFile(tb, relPaths...)
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		tb.Fatalf("failed to parse config %q: %v", path, err)
	}
	return cfg
}

func verifyTraceOutput(tb testing.TB, cfg *models.Config, res SimResult) {
	tb.Helper()
	s := engine.NewState(cfg)
	lastC := 0
	for _, step := range res.Steps {
		if step.Cycle < lastC {
			tb.Fatalf("step cycle %d went backwards from %d", step.Cycle, lastC)
		}
		s.AdvanceTo(step.Cycle)
		lastC = step.Cycle
		pIdx, ok := cfg.ProcIndex[step.Proc]
		if !ok {
			tb.Fatalf("unknown process %q in trace", step.Proc)
		}
		if err := s.Launch(pIdx); err != nil {
			tb.Fatalf("failed to launch %q at cycle %d: %v", step.Proc, step.Cycle, err)
		}
	}
	s.Drain()
}

func logEfficiency(t *testing.T, name string, cfg *models.Config, target string, res SimResult, elapsed time.Duration, note string) {
	t.Helper()
	idx := cfg.StockIndex[target]
	initQty := cfg.InitialStocks[idx]
	finalQty := res.Stocks[idx]
	evPerSec := 0.0
	if elapsed > 0 {
		evPerSec = float64(len(res.Steps)) / elapsed.Seconds()
	}
	t.Logf("[EFFICIENCY] %-10s | target=%s: %d -> %d (net %+d) | cycles=%d | steps=%d | time=%v (%.0f steps/s) | %s",
		name, target, initQty, finalQty, finalQty-initQty, res.LastCycle, len(res.Steps), elapsed, evPerSec, note)
}

func TestGreedy_Simple(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/simple", "resources/simple")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	if res.LastCycle != 60 {
		t.Errorf("expected optimal LastCycle 60, got %d", res.LastCycle)
	}
	clientIdx := cfg.StockIndex["client_content"]
	if res.Stocks[clientIdx] != 1 {
		t.Errorf("expected client_content == 1, got %d", res.Stocks[clientIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "simple", cfg, "client_content", res, elapsed, "100% Optimal (PDF reference: 60 cycles)")
}

func TestGreedy_Ikea(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/ikea", "resources/ikea")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	if res.LastCycle != 50 {
		t.Errorf("expected optimal LastCycle 50, got %d", res.LastCycle)
	}
	armoireIdx := cfg.StockIndex["armoire"]
	if res.Stocks[armoireIdx] != 1 {
		t.Errorf("expected armoire == 1, got %d", res.Stocks[armoireIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "ikea", cfg, "armoire", res, elapsed, "100% Optimal (PDF reference: 50 cycles)")
}

func TestGreedy_Steak(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/steak", "resources/steak")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	cuitIdx := cfg.StockIndex["steak_cuit"]
	if res.Stocks[cuitIdx] != 3 {
		t.Errorf("expected steak_cuit == 3, got %d", res.Stocks[cuitIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "steak", cfg, "steak_cuit", res, elapsed, "All 3 steaks cooked (40 cycles vs 30 theoretical min)")
}

func TestGreedy_Recre(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/recre", "resources/recre")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	marelleIdx := cfg.StockIndex["marelle"]
	if res.Stocks[marelleIdx] < 140 {
		t.Errorf("expected marelle >= 140 over 10000 cycles, got %d", res.Stocks[marelleIdx])
	}
	for _, step := range res.Steps {
		if strings.EqualFold(step.Proc, "manger") {
			t.Errorf("dead-end process manger was run in recre, but must be pruned")
		}
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "recre", cfg, "marelle", res, elapsed, "Self-sustaining cycle (~70 cycles/marelle, manger pruned)")
}

func TestGreedy_Pomme(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/pomme", "resources/pomme")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	euroIdx := cfg.StockIndex["euro"]
	initEuro := cfg.InitialStocks[euroIdx]
	if res.Stocks[euroIdx] <= initEuro {
		t.Errorf("expected net euro profit above initial %d, got %d", initEuro, res.Stocks[euroIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "pomme", cfg, "euro", res, elapsed, "Full pastry supply chain + vente_boite profit")
}

func TestGreedy_Inception(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/inception", "resources/inception")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	yearIdx := cfg.StockIndex["year"]
	if res.Stocks[yearIdx] < 10 {
		t.Errorf("expected year >= 10 over 10000 cycles, got %d", res.Stocks[yearIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "inception", cfg, "year", res, elapsed, "Catalyst clock preserved; dream:2 multiplier active")
}

func TestGreedy_Finite(t *testing.T) {
	cfg := loadConfig(t, "../test-config/finite.krp", "src/system/test-config/finite.krp")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	satIdx := cfg.StockIndex["orbital_satellite"]
	if res.Stocks[satIdx] != 2 {
		t.Errorf("expected orbital_satellite == 2, got %d", res.Stocks[satIdx])
	}
	for _, step := range res.Steps {
		if strings.EqualFold(step.Proc, "stamp_novelty_badge") {
			t.Errorf("stamp_novelty_badge was run in finite.krp, but must be excluded")
		}
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "finite", cfg, "orbital_satellite", res, elapsed, "100% Optimal (2 satellites in 215 cycles, trap pruned)")
}

func TestGreedy_Infinite(t *testing.T) {
	cfg := loadConfig(t, "../test-config/infinite.krp", "src/system/test-config/infinite.krp")
	start := time.Now()
	res := RunGreedy(context.Background(), cfg, 10000)
	elapsed := time.Since(start)

	bioIdx := cfg.StockIndex["biodiesel"]
	if res.Stocks[bioIdx] < 300 {
		t.Errorf("expected biodiesel >= 300 over 10000 cycles, got %d", res.Stocks[bioIdx])
	}
	verifyTraceOutput(t, cfg, res)
	logEfficiency(t, "infinite", cfg, "biodiesel", res, elapsed, "Closed-loop biorefinery recycling active")
}

func TestGreedy_ContextCancel(t *testing.T) {
	cfg := loadConfig(t, "../test-config/infinite.krp", "src/system/test-config/infinite.krp")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	res := RunGreedy(ctx, cfg, 100000)
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_PriorityUnit(t *testing.T) {
	cfg := loadConfig(t, "../test-config/finite.krp", "src/system/test-config/finite.krp")
	prio := ComputePriority(cfg)

	badgeIdx, ok := cfg.ProcIndex["stamp_novelty_badge"]
	if ok && prio.Useful[badgeIdx] {
		t.Errorf("expected stamp_novelty_badge to be marked useless")
	}

	satIdx, ok := cfg.ProcIndex["assemble_orbital_satellite"]
	if ok && !prio.Useful[satIdx] {
		t.Errorf("expected assemble_orbital_satellite to be marked useful")
	}

	for _, p := range prio.Order {
		if p == badgeIdx {
			t.Errorf("stamp_novelty_badge must not be in priority Order")
		}
	}
}

func runBenchConfig(b *testing.B, maxCycle int, paths ...string) {
	cfg := loadConfig(b, paths...)
	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()
	var totalSteps int
	for i := 0; i < b.N; i++ {
		res := RunGreedy(ctx, cfg, maxCycle)
		totalSteps += len(res.Steps)
	}
	if b.N > 0 {
		b.ReportMetric(float64(totalSteps)/float64(b.N), "steps/op")
	}
}

func BenchmarkGreedy_Simple(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/simple", "resources/simple")
}

func BenchmarkGreedy_Ikea(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/ikea", "resources/ikea")
}

func BenchmarkGreedy_Steak(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/steak", "resources/steak")
}

func BenchmarkGreedy_Recre(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/recre", "resources/recre")
}

func BenchmarkGreedy_Pomme(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/pomme", "resources/pomme")
}

func BenchmarkGreedy_Inception(b *testing.B) {
	runBenchConfig(b, 10000, "../../../resources/inception", "resources/inception")
}

func BenchmarkGreedy_Finite(b *testing.B) {
	runBenchConfig(b, 10000, "../test-config/finite.krp", "src/system/test-config/finite.krp")
}

func BenchmarkGreedy_Infinite(b *testing.B) {
	runBenchConfig(b, 10000, "../test-config/infinite.krp", "src/system/test-config/infinite.krp")
}
