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

func findConfigFile(t *testing.T, relPaths ...string) string {
	t.Helper()
	for _, p := range relPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	t.Fatalf("could not find file in paths: %v", relPaths)
	return ""
}

func loadConfig(t *testing.T, relPaths ...string) *models.Config {
	t.Helper()
	path := findConfigFile(t, relPaths...)
	cfg, err := parsing.ParseFile(path)
	if err != nil {
		t.Fatalf("failed to parse config %q: %v", path, err)
	}
	return cfg
}

func verifyTraceOutput(t *testing.T, cfg *models.Config, res SimResult) {
	t.Helper()
	s := engine.NewState(cfg)
	lastC := 0
	for _, step := range res.Steps {
		if step.Cycle < lastC {
			t.Fatalf("step cycle %d went backwards from %d", step.Cycle, lastC)
		}
		s.AdvanceTo(step.Cycle)
		lastC = step.Cycle
		pIdx, ok := cfg.ProcIndex[step.Proc]
		if !ok {
			t.Fatalf("unknown process %q in trace", step.Proc)
		}
		if err := s.Launch(pIdx); err != nil {
			t.Fatalf("failed to launch %q at cycle %d: %v", step.Proc, step.Cycle, err)
		}
	}
	s.Drain()
}

func TestGreedy_Simple(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/simple", "resources/simple")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	if res.LastCycle != 60 {
		t.Errorf("expected LastCycle 60, got %d", res.LastCycle)
	}
	clientIdx, ok := cfg.StockIndex["client_content"]
	if !ok || res.Stocks[clientIdx] < 1 {
		t.Errorf("expected client_content >= 1, got %d", res.Stocks[clientIdx])
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Ikea(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/ikea", "resources/ikea")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	if res.LastCycle != 50 {
		t.Errorf("expected LastCycle 50, got %d", res.LastCycle)
	}
	armoireIdx, ok := cfg.StockIndex["armoire"]
	if !ok || res.Stocks[armoireIdx] < 1 {
		t.Errorf("expected armoire >= 1, got %d", res.Stocks[armoireIdx])
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Steak(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/steak", "resources/steak")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	cuitIdx, ok := cfg.StockIndex["steak_cuit"]
	if !ok || res.Stocks[cuitIdx] != 3 {
		t.Errorf("expected steak_cuit == 3, got %d", res.Stocks[cuitIdx])
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Recre(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/recre", "resources/recre")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	marelleIdx, ok := cfg.StockIndex["marelle"]
	if !ok || res.Stocks[marelleIdx] < 2 {
		t.Errorf("expected marelle >= 2, got %d", res.Stocks[marelleIdx])
	}
	for _, step := range res.Steps {
		if strings.EqualFold(step.Proc, "manger") {
			t.Errorf("manger was run in recre, but must be excluded")
		}
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Finite(t *testing.T) {
	cfg := loadConfig(t, "../test-config/finite.krp", "src/system/test-config/finite.krp")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	satIdx, ok := cfg.StockIndex["orbital_satellite"]
	if !ok || res.Stocks[satIdx] != 2 {
		t.Errorf("expected orbital_satellite == 2, got %d", res.Stocks[satIdx])
	}
	for _, step := range res.Steps {
		if strings.EqualFold(step.Proc, "stamp_novelty_badge") {
			t.Errorf("stamp_novelty_badge was run in finite.krp, but must be excluded")
		}
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Infinite(t *testing.T) {
	cfg := loadConfig(t, "../test-config/infinite.krp", "src/system/test-config/infinite.krp")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 500)

	bioIdx, ok := cfg.StockIndex["biodiesel"]
	if !ok || res.Stocks[bioIdx] <= 0 {
		t.Errorf("expected biodiesel > 0, got %d", res.Stocks[bioIdx])
	}
	if res.LastCycle < 100 {
		t.Errorf("expected multiple cycles (> 100), got LastCycle %d", res.LastCycle)
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Inception(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/inception", "resources/inception")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 2000)

	yearIdx, ok := cfg.StockIndex["year"]
	if !ok || res.Stocks[yearIdx] < 1 {
		t.Errorf("expected year >= 1, got %d", res.Stocks[yearIdx])
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_Pomme(t *testing.T) {
	cfg := loadConfig(t, "../../../resources/pomme", "resources/pomme")
	ctx := context.Background()
	res := RunGreedy(ctx, cfg, 1000)

	euroIdx, ok := cfg.StockIndex["euro"]
	if !ok || res.Stocks[euroIdx] <= 0 {
		t.Errorf("expected euro > 0, got %d", res.Stocks[euroIdx])
	}
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_ContextCancel(t *testing.T) {
	cfg := loadConfig(t, "../config/infinite.krp", "src/system/config/infinite.krp")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	res := RunGreedy(ctx, cfg, 100000)
	verifyTraceOutput(t, cfg, res)
}

func TestGreedy_PriorityUnit(t *testing.T) {
	cfg := loadConfig(t, "../config/finite.krp", "src/system/config/finite.krp")
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
