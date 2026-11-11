package solver

import (
	"krpsim/internal/config"
	"krpsim/internal/sim"
)

// A catalyst is a stock a process needs and gives back unchanged (precision_assembler, poele).
// A root catalyst is one that already exists at the start; the others have to be built.

type planner struct {
	cfg   *config.Config
	st    *sim.State
	order []int

	isTarget  []bool
	isCat     []bool
	isRootCat []bool
	builtCats []int

	borrows   [][]int // per process: catalysts it uses
	gains     [][]int // per process: stocks it ends up with more of
	borrowers [][]int // per stock: processes using it as a catalyst
	breeders  [][]int // per stock: processes that consume it and still net-produce it

	outpaced []bool // a faster catalyst-based process makes the same output
	demand   []int  // largest single-run need of a stock among processes worth feeding
	reserve  int

	inFlight []int
}

func newPlanner(cfg *config.Config, st *sim.State) *planner {
	nStocks, nProcs := len(cfg.StockNames), len(cfg.Processes)
	pl := &planner{
		cfg:       cfg,
		st:        st,
		isTarget:  make([]bool, nStocks),
		isCat:     make([]bool, nStocks),
		isRootCat: make([]bool, nStocks),
		borrows:   make([][]int, nProcs),
		gains:     make([][]int, nProcs),
		borrowers: make([][]int, nStocks),
		breeders:  make([][]int, nStocks),
		outpaced:  make([]bool, nProcs),
		demand:    make([]int, nStocks),
		inFlight:  make([]int, nStocks),
	}
	for _, t := range cfg.Targets {
		pl.isTarget[t] = true
	}

	maxBatch := 0
	for p := range cfg.Processes {
		proc := &cfg.Processes[p]
		for _, n := range proc.Needs {
			maxBatch = max(maxBatch, n.Qty)
			switch net := proc.Net[n.Stock]; {
			case net == 0:
				pl.borrows[p] = append(pl.borrows[p], n.Stock)
				pl.borrowers[n.Stock] = append(pl.borrowers[n.Stock], p)
				pl.isCat[n.Stock] = true
				if cfg.InitialStocks[n.Stock] > 0 {
					pl.isRootCat[n.Stock] = true
				}
			case net > 0:
				pl.breeders[n.Stock] = append(pl.breeders[n.Stock], p)
			}
		}
		for s, net := range proc.Net {
			if net > 0 {
				pl.gains[p] = append(pl.gains[p], s)
			}
		}
	}
	pl.reserve = maxBatch + reserveSlack

	for s := range pl.isCat {
		if pl.isCat[s] && !pl.isRootCat[s] {
			pl.builtCats = append(pl.builtCats, s)
		}
	}

	order, useful := rank(cfg)
	pl.order = order
	for p := range cfg.Processes {
		pl.outpaced[p] = pl.isOutpaced(p)
		if !useful[p] || pl.outpaced[p] {
			continue
		}
		for _, n := range cfg.Processes[p].Needs {
			pl.demand[n.Stock] = max(pl.demand[n.Stock], n.Qty)
		}
	}
	return pl
}

// isOutpaced is true for a process without catalysts when some faster
// catalyst-based process produces one of the same outputs.
func (pl *planner) isOutpaced(p int) bool {
	if len(pl.borrows[p]) > 0 {
		return false
	}
	delay := pl.cfg.Processes[p].Delay
	for _, s := range pl.gains[p] {
		for q := range pl.cfg.Processes {
			other := &pl.cfg.Processes[q]
			if q != p && other.Net[s] > 0 && len(pl.borrows[q]) > 0 && other.Delay < delay {
				return true
			}
		}
	}
	return false
}
