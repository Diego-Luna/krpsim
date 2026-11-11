package sim

// InFlight fills dst with what the running processes will hand back.
func (s *State) InFlight(dst []int) {
	clear(dst)
	for _, ev := range s.events {
		for _, r := range s.Cfg.Processes[ev.proc].Results {
			dst[r.Stock] += r.Qty
		}
	}
}
