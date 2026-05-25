package bbc

import "errors"

// ErrNoOSROM is returned by Reset, ColdReset, and the first Tick
// after construction when no OS ROM image has been loaded.
var ErrNoOSROM = errors.New("bbc: no OS ROM loaded")

// ErrInvalidROMSize is returned by LoadOSROM and LoadSidewaysROM
// when the supplied image is not exactly 16384 bytes.
var ErrInvalidROMSize = errors.New("bbc: ROM image must be exactly 16384 bytes")

// ErrBankOutOfRange is returned by LoadSidewaysROM when bank is
// outside 0..3.
var ErrBankOutOfRange = errors.New("bbc: sideways bank index must be 0..3")

// ErrRestoreMismatch is returned by Restore when the snapshot's
// loaded-bank set differs from the destination Machine's.
var ErrRestoreMismatch = errors.New("bbc: restored snapshot has different sideways banks loaded than the current machine")
