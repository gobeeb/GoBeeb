package mos6502

// BusEventKind is the direction of a single bus access.
type BusEventKind uint8

// BusRead / BusWrite are the two possible BusEventKind values.
const (
	BusRead BusEventKind = iota
	BusWrite
)

// BusEvent is a single recorded bus access: cycle counter at the time
// of the access, the address on the bus, the byte transferred, and the
// direction.
type BusEvent struct {
	Cycle uint64
	Addr  uint16
	Value uint8
	Kind  BusEventKind
}

// Trace is an optional, fixed-capacity ring buffer that records every
// bus cycle the CPU performs while attached via (*CPU).SetTrace. (SC-008)
//
// Trace itself does no synchronisation; a Trace is owned by the CPU it
// is attached to.
type Trace struct {
	events []BusEvent
	head   int
	full   bool
}

// NewTrace returns a pre-allocated Trace with the given capacity. A
// capacity ≤ 0 is treated as 1.
func NewTrace(capacity int) *Trace {
	if capacity <= 0 {
		capacity = 1
	}
	return &Trace{events: make([]BusEvent, capacity)}
}

// append records one bus event. Zero allocation in steady state — the
// backing slice is pre-allocated by NewTrace.
func (t *Trace) append(e BusEvent) {
	t.events[t.head] = e
	t.head++
	if t.head == len(t.events) {
		t.head = 0
		t.full = true
	}
}

// Snapshot returns the recorded events in chronological order. The
// returned slice is a freshly allocated copy and is safe for the caller
// to retain or mutate.
func (t *Trace) Snapshot() []BusEvent {
	if !t.full {
		out := make([]BusEvent, t.head)
		copy(out, t.events[:t.head])
		return out
	}
	out := make([]BusEvent, len(t.events))
	copy(out, t.events[t.head:])
	copy(out[len(t.events)-t.head:], t.events[:t.head])
	return out
}

// Len reports the number of events currently retained.
func (t *Trace) Len() int {
	if t.full {
		return len(t.events)
	}
	return t.head
}

// Reset clears the trace without re-allocating the backing buffer.
func (t *Trace) Reset() {
	t.head = 0
	t.full = false
}
