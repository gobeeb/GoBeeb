//go:build ignore

// Package mos6502 contract definitions.
//
// This file is the public-interface contract for the host-supplied memory /
// address-bus collaborator. It mirrors what will be implemented in
// `mos6502/memory.go` once Phase 2 (`/speckit-tasks`) begins. Spec references:
// FR-006, FR-018, FR-023; Research §2 (infallibility).

package mos6502

// Memory is the host-supplied collaborator that backs every bus cycle the CPU
// performs. The CPU never holds its own copy of the address space: every read
// and every write — instruction fetch, operand fetch, indirect-pointer fetch,
// effective-address access, stack push/pull, vector fetch, dummy reads on
// indexed page-cross, and the dummy write during read-modify-write — is
// delegated to this interface in the cycle order a real NMOS 6502 issues
// them.
//
// The interface is intentionally infallible:
//
//   - Real NMOS silicon has no concept of a bus error; every cycle, an
//     address is on the bus and a byte either flows in (read) or out
//     (write). There is no protocol path for the bus to signal failure
//     back into the CPU.
//   - Returning an error per cycle would force allocation on the hot
//     path (boxing into the error interface) and a per-cycle branch,
//     both fatal to the SC-006 performance budget of ≤ 125 ns/cycle.
//   - The host has full freedom to handle "internal" failures (unmapped
//     address, ROM bank fault, peripheral error) in its own state and
//     simply return whatever byte a real BBC bus would put there — by
//     convention $FF for open-bus reads — to the CPU.
//
// Implementations of Memory MUST be deterministic with respect to the
// inputs the CPU presents: given the same sequence of Read and Write
// calls, an implementation MUST produce the same sequence of returned
// bytes and the same final internal state. Non-determinism in Memory is
// the single biggest cause of non-determinism in the overall emulator
// (SC-007) and is strongly discouraged.
//
// Implementations are NOT required to be safe for concurrent use. A
// single CPU instance accesses its Memory from one goroutine at a time
// (Assumption A8 in spec.md). Sharing a Memory between multiple CPU
// instances or between the CPU and a renderer thread is the host's
// responsibility to serialise.
//
// Implementations MUST handle every address in the 16-bit range
// [$0000, $FFFF]. The CPU does not bounds-check addresses and will pass
// any 16-bit value through. ROM regions, memory-mapped I/O regions,
// and unmapped regions are all the host's concern.
type Memory interface {
	// Read returns the byte at addr. Called once per CPU bus cycle in
	// which the CPU asserts the read line. For memory-mapped I/O
	// peripherals that have read side effects (e.g. clearing a status
	// register on read), Read is the side-effect site.
	Read(addr uint16) uint8

	// Write stores value at addr. Called once per CPU bus cycle in
	// which the CPU asserts the write line. For ROM regions, Write
	// MAY silently discard the value (matching real BBC hardware).
	// For memory-mapped I/O peripherals, Write is the command-issue
	// site.
	Write(addr uint16, value uint8)
}
