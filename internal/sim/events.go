package sim

type event struct {
	end  int
	proc int
}

// eventHeap is a min-heap on end cycle; ties go to the lower process index so runs are reproducible.
type eventHeap []event

func (h eventHeap) less(i, j int) bool {
	if h[i].end != h[j].end {
		return h[i].end < h[j].end
	}
	return h[i].proc < h[j].proc
}

func (h *eventHeap) push(ev event) {
	*h = append(*h, ev)
	h.up(len(*h) - 1)
}

func (h *eventHeap) pop() event {
	old := *h
	n := len(old)
	if n == 0 {
		return event{}
	}
	top := old[0]
	old[0] = old[n-1]
	*h = old[:n-1]
	h.down(0)
	return top
}

func (h eventHeap) up(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !h.less(i, parent) {
			break
		}
		h[i], h[parent] = h[parent], h[i]
		i = parent
	}
}

func (h eventHeap) down(i int) {
	n := len(h)
	for {
		best := 2*i + 1
		if best >= n {
			return
		}
		if right := best + 1; right < n && h.less(right, best) {
			best = right
		}
		if !h.less(best, i) {
			return
		}
		h[i], h[best] = h[best], h[i]
		i = best
	}
}
