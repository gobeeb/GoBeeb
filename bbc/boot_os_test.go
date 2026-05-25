package bbc

import (
	"os"
	"testing"
)

// TestBootRealOSROM is the SC-001 smoke test. Gated on $BBC_OS_ROM
// because the project does not redistribute the copyrighted OS 1.20
// image. Set BBC_OS_ROM=/path/to/os120.rom to enable.
func TestBootRealOSROM(t *testing.T) {
	path := os.Getenv("BBC_OS_ROM")
	if path == "" {
		t.Skip("BBC_OS_ROM not set; skipping real-ROM smoke test")
	}
	image, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read OS ROM %q: %v", path, err)
	}

	m := New()
	if err := m.LoadOSROM(image); err != nil {
		t.Fatalf("LoadOSROM: %v", err)
	}

	var unmapped, illegal int
	m.SetUnmappedAccessHook(func(addr uint16, write bool, value uint8) {
		unmapped++
		if unmapped <= 8 {
			kind := "read"
			if write {
				kind = "write"
			}
			t.Logf("unmapped %s $%04X value=$%02X", kind, addr, value)
		}
	})
	m.CPU().SetIllegalOpcodeHook(func(pc uint16, opcode uint8) {
		illegal++
		if illegal <= 8 {
			t.Logf("illegal opcode %#02X at $%04X", opcode, pc)
		}
	})

	if err := m.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	m.Tick(1_000_000)

	if unmapped != 0 {
		t.Errorf("OS ROM boot fired UnmappedAccessHook %d times; want 0", unmapped)
	}
	if illegal != 0 {
		t.Errorf("OS ROM boot fired IllegalOpcodeHook %d times; want 0", illegal)
	}
}
