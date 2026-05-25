package bbc

// romImageSize is the fixed BBC ROM image size (16 KB).
const romImageSize = 0x4000

// RomBanks owns the OS ROM image plus the four sideways ROM bank
// slots and the latched bank index. All bytes are owned by the
// Machine after load (Clarification Q4): callers may mutate, reuse,
// or release the input slice immediately.
type RomBanks struct {
	os       [romImageSize]byte
	osLoaded bool

	sideways       [4][romImageSize]byte
	sidewaysLoaded [4]bool

	bank uint8 // currently-selected bank index (0..3)
}

// LoadOSROM installs the 16 KB OS ROM image at $C000–$FBFF +
// $FF00–$FFFF. The image MUST be exactly 16384 bytes; otherwise
// ErrInvalidROMSize is returned and no state is modified. The
// image is copied into internally-owned storage.
//
// Machine state (CPU registers, RAM, peripheral registers) is
// preserved across calls; the caller is responsible for issuing
// Reset() if guest software is running.
func (m *Machine) LoadOSROM(image []byte) error {
	if len(image) != romImageSize {
		return ErrInvalidROMSize
	}
	copy(m.rom.os[:], image)
	m.rom.osLoaded = true
	return nil
}

// LoadSidewaysROM installs a 16 KB ROM image into bank (0..3). The
// image MUST be exactly 16384 bytes; otherwise ErrInvalidROMSize is
// returned. bank outside 0..3 returns ErrBankOutOfRange. The image
// is copied into internally-owned storage.
//
// Loading does not change the latched bank index ($FE30). MOS code
// or the host must write to $FE30 to make the new bank visible at
// $8000–$BFFF.
func (m *Machine) LoadSidewaysROM(bank int, image []byte) error {
	if bank < 0 || bank > 3 {
		return ErrBankOutOfRange
	}
	if len(image) != romImageSize {
		return ErrInvalidROMSize
	}
	copy(m.rom.sideways[bank][:], image)
	m.rom.sidewaysLoaded[bank] = true
	return nil
}
