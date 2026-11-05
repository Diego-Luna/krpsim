package engine

// ? maybe change the type to the fodlers model
// Event represents a process completion event scheduled at a given cycle
type Event struct {
	End  int
	Proc int
}

// eventHeap is a min-heap of Event items ordered primarily by End cycle
// Ties are broken deterministically by Proc index
type eventHeap []Event

func (h eventHeap) less(i, j int) bool {
	if h[i].End != h[j].End {
		return h[i].End < h[j].End
	}
	return h[i].Proc < h[j].Proc
}

func (h *eventHeap) push(ev Event) {
	*h = append(*h, ev)
	h.siftUp(len(*h) - 1)
}

func (h *eventHeap) pop() Event {
	old := *h
	n := len(old)
	if n == 0 {
		return Event{}
	}
	top := old[0]
	old[0] = old[n-1]
	*h = old[:n-1]
	if len(*h) > 0 {
		h.siftDown(0)
	}
	return top
}

func (h *eventHeap) peek() (Event, bool) {
	if len(*h) == 0 {
		return Event{}, false
	}
	return (*h)[0], true
}

func (h eventHeap) siftUp(i int) {
	for i > 0 {
		parent := (i - 1) / 2
		if !h.less(i, parent) {
			break
		}
		h[i], h[parent] = h[parent], h[i]
		i = parent
	}
}

func (h eventHeap) siftDown(i int) {
	n := len(h)
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		best := left
		right := left + 1
		if right < n && h.less(right, left) {
			best = right
		}
		if !h.less(best, i) {
			break
		}
		h[i], h[best] = h[best], h[i]
		i = best
	}
}
