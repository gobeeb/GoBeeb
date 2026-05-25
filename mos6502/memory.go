package mos6502

// Memory is the host-supplied collaborator that backs every bus cycle the
// CPU performs. The CPU never holds its own copy of the address space:
// every read and every write — instruction fetch, operand fetch,
// indirect-pointer fetch, effective-address access, stack push/pull,
// vector fetch, dummy reads on indexed page-cross, and the dummy write
// during read-modify-write — is delegated to this interface in the cycle
// order a real NMOS 6502 issues them. (FR-006, FR-018, FR-023)
//
// The interface is intentionally infallible. Real NMOS silicon has no
// concept of a bus error; there is no protocol path for the bus to
// signal failure back into the CPU. Returning error per cycle would
// force allocation on the hot path and a per-cycle branch, both fatal
// to the SC-006 performance budget. The host is free to handle
// "internal" failures (unmapped address, ROM bank fault, peripheral
// error) in its own state and return whatever byte a real BBC bus would
// put there (by convention $FF for open-bus reads).
//
// Implementations of Memory MUST be deterministic with respect to the
// inputs the CPU presents (SC-007). Implementations are NOT required to
// be safe for concurrent use; a single CPU instance accesses its Memory
// from one goroutine at a time (Assumption A8).
//
// Implementations MUST handle every address in the 16-bit range
// [$0000, $FFFF]. The CPU does not bounds-check addresses.
type Memory interface {
	// Read returns the byte at addr. Called once per CPU bus cycle in
	// which the CPU asserts the read line.
	Read(addr uint16) uint8

	// Write stores value at addr. Called once per CPU bus cycle in
	// which the CPU asserts the write line.
	Write(addr uint16, value uint8)
}
