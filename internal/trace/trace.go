package trace

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	mascotOpen  = "【"
	Mascot      = mascotOpen + "=◈︿◈=】"
	Evaluating  = "Evaluating .................. done."
	Finished    = "no more process doable at time"
	StockHeader = "Stock :"
	StockArrow  = "=>"
)

// Lines starting with these (case-insensitive) are banner output, ours or the subject's reference krpsim.
var bannerPrefixes = []string{"file", "evaluating", "main walk", "no more process"}

type Step struct {
	Cycle int
	Proc  string
}

func (s Step) String() string {
	return strconv.Itoa(s.Cycle) + ":" + s.Proc
}

func isBanner(line string) bool {
	if strings.Contains(line, StockArrow) || strings.HasPrefix(line, ">") || strings.HasPrefix(line, mascotOpen) {
		return true
	}
	lower := strings.ToLower(line)
	for _, p := range bannerPrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return lower == strings.ToLower(StockHeader) || lower == "stock:"
}

// Parse reads one line of a trace. skip is true for blank, comment and banner lines.
func Parse(raw string) (st Step, skip bool, err error) {
	line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
	if line == "" || isBanner(line) {
		return Step{}, true, nil
	}
	sep := strings.IndexByte(line, ':')
	if sep <= 0 || sep == len(line)-1 {
		return Step{}, false, fmt.Errorf("malformed trace line %q: expected <cycle>:<process>", line)
	}
	cycleStr := strings.TrimSpace(line[:sep])
	proc := strings.TrimSpace(line[sep+1:])
	if proc == "" {
		return Step{}, false, fmt.Errorf("malformed trace line %q: empty process name", line)
	}
	cycle, err := strconv.Atoi(cycleStr)
	if err != nil || cycle < 0 {
		return Step{Proc: proc}, false, fmt.Errorf("invalid cycle %q in trace line %q", cycleStr, line)
	}
	return Step{Cycle: cycle, Proc: proc}, false, nil
}
