package config

type StockDelta struct {
	Stock int
	Qty   int
}

type Process struct {
	Name    string
	Needs   []StockDelta
	Results []StockDelta
	Delay   int

	// Dense views indexed by stock id, filled once the whole file is parsed.
	NeedQty []int
	Net     []int
}

type Config struct {
	StockNames    []string
	StockIndex    map[string]int
	InitialStocks []int
	Processes     []Process
	ProcIndex     map[string]int
	OptimizeTime  bool
	Targets       []int
}

func (c *Config) finalize() {
	n := len(c.StockNames)
	for i := range c.Processes {
		p := &c.Processes[i]
		p.NeedQty = make([]int, n)
		p.Net = make([]int, n)
		for _, d := range p.Needs {
			p.NeedQty[d.Stock] += d.Qty
			p.Net[d.Stock] -= d.Qty
		}
		for _, d := range p.Results {
			p.Net[d.Stock] += d.Qty
		}
	}
}
