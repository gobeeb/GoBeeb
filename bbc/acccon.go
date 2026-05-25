package bbc

// ACCCON is the ACCCON / paged-ROM-ID stub. Model B has no
// ACCCON: reads return open-bus $FF, writes are silently dropped.
// The stub exists so Master 128 work in a future phase can drop
// in real behaviour without changing the decoder.
type ACCCON struct{}

// Read implements Peripheral; always returns $FF on Model B.
func (a *ACCCON) Read(_ uint8) uint8 { return openBus }

// Write implements Peripheral; no-op on Model B.
func (a *ACCCON) Write(_ uint8, _ uint8) {}

// Zero is a no-op (ACCCON carries no register state on Model B).
func (a *ACCCON) Zero() {}
