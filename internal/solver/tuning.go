package solver

const (
	targetValue  = 1000.0 // worth of one unit of an optimize target
	neutralValue = 1.0    // worth of every stock when only time is optimized
	valuePasses  = 15     // relaxation rounds when pushing value back to inputs
	valueDecay   = 1.5    // value lost per step away from a target
	inputPenalty = 0.3    // share of the consumed value subtracted from a score
	reserveSlack = 5      // extra raw stock to collect before spending a root catalyst

	maxLaunchesPerCycle = 1000
	defaultMaxCycle     = 10000
)
