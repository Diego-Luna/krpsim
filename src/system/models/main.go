package models

type StockDelta struct {
	Stock int
	Qty   int
}

type Process struct {
	Name    string
	Needs   []StockDelta
	Results []StockDelta
	Delay   int
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
