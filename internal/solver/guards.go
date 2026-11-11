package solver

import "krpsim/internal/config"

// blocked reports whether starting p now would waste something a better
// plan still needs: a catalyst, a reserve of raw stock, or demand already met.
func (pl *planner) blocked(p int) bool {
	return pl.guardsCatalyst(p) || pl.capped(p)
}

func (pl *planner) have(s int) int {
	return pl.st.Stocks[s] + pl.inFlight[s]
}

func (pl *planner) guardsCatalyst(p int) bool {
	if pl.outpaced[p] {
		for _, c := range pl.builtCats {
			if pl.have(c) > 0 {
				return true
			}
		}
	}
	proc := &pl.cfg.Processes[p]
	for _, n := range proc.Needs {
		s, qty := n.Stock, n.Qty
		switch net := proc.Net[s]; {
		case net < 0 && pl.isRootCat[s]:
			if pl.rootCatTooEarly(s, qty) {
				return true
			}
		case net < 0 && pl.isCat[s]:
			if pl.catalystStillWanted(s, qty) {
				return true
			}
		case net == 0:
			if pl.have(s) < pl.demand[s] && qty < pl.demand[s] && pl.breederCanRun(proc, s) {
				return true
			}
		}
	}
	return false
}

// catalystStillWanted: eating built catalyst s would starve a process that could use it right now.
func (pl *planner) catalystStillWanted(s, qty int) bool {
	for _, q := range pl.borrowers[s] {
		user := &pl.cfg.Processes[q]
		if user.NeedQty[s] >= qty && pl.inputsCovered(user, s) {
			return true
		}
	}
	return false
}

// rootCatTooEarly: spending root catalyst s now, before its users built up a reserve.
func (pl *planner) rootCatTooEarly(s, qty int) bool {
	if pl.st.Stocks[s] > qty {
		return false
	}
	for _, q := range pl.borrowers[s] {
		for _, out := range pl.gains[q] {
			if pl.have(out) < pl.reserve {
				return true
			}
		}
	}
	return false
}

// breederCanRun: p borrows catalyst s while some process that multiplies s is ready to go.
func (pl *planner) breederCanRun(proc *config.Process, s int) bool {
	for _, q := range pl.breeders[s] {
		for _, n := range pl.cfg.Processes[q].Needs {
			if n.Stock != s && (proc.Net[n.Stock] <= 0 || pl.have(n.Stock) >= n.Qty) {
				return true
			}
		}
	}
	return false
}

func (pl *planner) inputsCovered(proc *config.Process, except int) bool {
	for _, n := range proc.Needs {
		if n.Stock != except && pl.have(n.Stock) < n.Qty {
			return false
		}
	}
	return true
}

// capped: every intermediate p makes is already stocked up to demand, so running it is pointless.
func (pl *planner) capped(p int) bool {
	if len(pl.cfg.Targets) == 0 {
		return false
	}
	for _, s := range pl.borrows[p] {
		if !pl.isRootCat[s] {
			continue
		}
		for _, out := range pl.gains[p] {
			if pl.have(out) < pl.reserve {
				return false
			}
		}
	}
	intermediate, met := false, true
	for _, s := range pl.gains[p] {
		if pl.isTarget[s] {
			return false
		}
		if d := pl.demand[s]; d > 0 {
			intermediate = true
			if pl.have(s) < d {
				met = false
			}
		}
	}
	return intermediate && met
}
