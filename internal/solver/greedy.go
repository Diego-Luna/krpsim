package solver

import (
	"context"

	"krpsim/internal/config"
	"krpsim/internal/sim"
	"krpsim/internal/trace"
)

type Result struct {
	Steps     []trace.Step
	LastCycle int
	Stocks    []int
}

// Run plays the greedy schedule until nothing can start, maxCycle is
// passed (0 picks the default) or ctx runs out.
func Run(ctx context.Context, cfg *config.Config, maxCycle int) Result {
	if maxCycle <= 0 {
		maxCycle = defaultMaxCycle
	}
	st := sim.NewState(cfg)
	pl := newPlanner(cfg, st)

	var steps []trace.Step
	for ctx.Err() == nil && st.Cycle <= maxCycle {
		steps = pl.launchAll(steps)
		next, ok := st.NextEvent()
		if !ok || next > maxCycle || ctx.Err() != nil {
			break
		}
		st.AdvanceTo(next)
	}
	st.Drain()
	return Result{Steps: steps, LastCycle: st.Cycle, Stocks: append([]int(nil), st.Stocks...)}
}

// launchAll starts processes one at a time, best first, until nothing else is allowed this cycle.
func (pl *planner) launchAll(steps []trace.Step) []trace.Step {
	for range maxLaunchesPerCycle {
		p, ok := pl.pick()
		if !ok {
			break
		}
		if err := pl.st.Launch(p); err != nil {
			panic(err) // pick only returns launchable processes
		}
		steps = append(steps, trace.Step{Cycle: pl.st.Cycle, Proc: pl.cfg.Processes[p].Name})
	}
	return steps
}

func (pl *planner) pick() (int, bool) {
	pl.st.InFlight(pl.inFlight)
	for _, p := range pl.order {
		if pl.st.CanLaunch(p) && !pl.blocked(p) {
			return p, true
		}
	}
	return 0, false
}
