package verify

import (
	"bufio"
	"fmt"
	"io"

	"krpsim/internal/config"
	"krpsim/internal/sim"
	"krpsim/internal/trace"
)

type Result struct {
	OK        bool
	ErrCycle  int
	ErrProc   string
	ErrReason string
	LastCycle int
	Stocks    []int
}

// Replay checks a krpsim trace read from r. The error is only for I/O failures;
// an invalid trace is reported through Result.
func Replay(cfg *config.Config, r io.Reader) (Result, error) {
	st := sim.NewState(cfg)
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		step, skip, err := trace.Parse(sc.Text())
		if err != nil {
			return failed(st, st.Cycle, step.Proc, err.Error()), nil
		}
		if skip {
			continue
		}
		if reason := apply(cfg, st, step); reason != "" {
			return failed(st, step.Cycle, step.Proc, reason), nil
		}
	}
	if err := sc.Err(); err != nil {
		return Result{}, err
	}
	return finished(st), nil
}

func Steps(cfg *config.Config, steps []trace.Step) Result {
	st := sim.NewState(cfg)
	for _, step := range steps {
		if reason := apply(cfg, st, step); reason != "" {
			return failed(st, step.Cycle, step.Proc, reason)
		}
	}
	return finished(st)
}

func apply(cfg *config.Config, st *sim.State, step trace.Step) string {
	if step.Cycle < st.Cycle {
		return fmt.Sprintf("cycle %d is before current cycle %d", step.Cycle, st.Cycle)
	}
	st.AdvanceTo(step.Cycle)
	p, ok := cfg.ProcIndex[step.Proc]
	if !ok {
		return "unknown process"
	}
	if err := st.Launch(p); err != nil {
		return err.Error()
	}
	return ""
}

func failed(st *sim.State, cycle int, proc, reason string) Result {
	return Result{
		ErrCycle:  cycle,
		ErrProc:   proc,
		ErrReason: reason,
		LastCycle: st.Cycle,
		Stocks:    append([]int(nil), st.Stocks...),
	}
}

func finished(st *sim.State) Result {
	st.Drain()
	return Result{OK: true, LastCycle: st.Cycle, Stocks: append([]int(nil), st.Stocks...)}
}
