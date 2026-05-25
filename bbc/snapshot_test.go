package bbc

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Acceptance scenario 1: two Machine instances loaded with the
// same OS ROM converge to bit-identical CPU+RAM state after
// Snapshot→Restore→Tick.
func TestSnapshot_RoundTripCPUAndRAM(t *testing.T) {
	a := newWithStubOS(t)
	if err := a.Reset(); err != nil {
		t.Fatalf("a.Reset: %v", err)
	}
	a.Tick(150_000) // >= 100k cycles per SC-005

	snap := a.Snapshot()

	b := newWithStubOS(t)
	if err := b.Restore(snap); err != nil {
		t.Fatalf("b.Restore: %v", err)
	}

	const more = 50_000
	a.Tick(more)
	b.Tick(more)

	if a.cpu.Registers() != b.cpu.Registers() {
		t.Fatalf("CPU regs diverged: a=%+v b=%+v", a.cpu.Registers(), b.cpu.Registers())
	}
	if a.mmap.ram != b.mmap.ram {
		t.Fatal("RAM diverged after Snapshot/Restore/Tick")
	}
}

// Acceptance scenario 2: sideways bank index survives the
// round-trip — restored machine reads $8000 from the same bank
// without re-writing $FE30.
func TestSnapshot_PreservesSidewaysBank(t *testing.T) {
	a := newWithStubOS(t)
	_ = a.LoadSidewaysROM(0, stubSidewaysAA)
	_ = a.LoadSidewaysROM(3, stubSideways55)
	a.mmap.Write(0xFE30, 0x03) // select bank 3

	snap := a.Snapshot()

	b := newWithStubOS(t)
	_ = b.LoadSidewaysROM(0, stubSidewaysAA)
	_ = b.LoadSidewaysROM(3, stubSideways55)
	if err := b.Restore(snap); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := b.mmap.Read(0x8000); got != 0x55 {
		t.Fatalf("restored bank 3 read $8000 = $%02X, want $55", got)
	}
}

// Acceptance scenario 3: System VIA stub state survives.
func TestSnapshot_PreservesVIARegisters(t *testing.T) {
	a := newWithStubOS(t)
	a.mmap.Write(0xFE40, 0x11)
	a.mmap.Write(0xFE4F, 0xEE)
	a.mmap.Write(0xFE60, 0x22) // user VIA too
	a.mmap.Write(0xFE00, 0x05) // CRTC index = 5
	a.mmap.Write(0xFE01, 0xC3) // CRTC R5 = $C3

	snap := a.Snapshot()

	b := newWithStubOS(t)
	if err := b.Restore(snap); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if got := b.mmap.Read(0xFE40); got != 0x11 {
		t.Errorf("System VIA reg 0 = $%02X, want $11", got)
	}
	if got := b.mmap.Read(0xFE4F); got != 0xEE {
		t.Errorf("System VIA reg 15 = $%02X, want $EE", got)
	}
	if got := b.mmap.Read(0xFE60); got != 0x22 {
		t.Errorf("User VIA reg 0 = $%02X, want $22", got)
	}
	// CRTC: re-select R5, read data port; should be $C3.
	b.mmap.Write(0xFE00, 0x05)
	if got := b.mmap.Read(0xFE01); got != 0xC3 {
		t.Errorf("CRTC R5 = $%02X, want $C3", got)
	}
}

// Restore returns ErrNoOSROM on a Machine constructed without
// LoadOSROM.
func TestRestore_NoOSROM(t *testing.T) {
	m := New()
	err := m.Restore(Snapshot{})
	if !errors.Is(err, ErrNoOSROM) {
		t.Fatalf("got %v, want ErrNoOSROM", err)
	}
}

// Restore returns ErrRestoreMismatch when the loaded-bank set
// differs between snapshot and destination.
func TestRestore_BankMismatch(t *testing.T) {
	a := newWithStubOS(t)
	_ = a.LoadSidewaysROM(0, stubSidewaysAA)
	snap := a.Snapshot()

	b := newWithStubOS(t)
	// b has bank 1 loaded, but snap recorded only bank 0.
	_ = b.LoadSidewaysROM(1, stubSideways55)
	err := b.Restore(snap)
	if !errors.Is(err, ErrRestoreMismatch) {
		t.Fatalf("got %v, want ErrRestoreMismatch", err)
	}
}

// FR-024: Snapshot must NOT contain OS-ROM or sideways-ROM image
// bytes. Verify via reflection that no field on Snapshot or
// PeripheralSnapshot is a 16384-byte array.
func TestSnapshot_OmitsROMImages(t *testing.T) {
	walk := func(t *testing.T, root reflect.Type) {
		t.Helper()
		var visit func(typ reflect.Type, path string)
		visit = func(typ reflect.Type, path string) {
			if typ.Kind() == reflect.Array {
				if typ.Len() == romImageSize {
					t.Errorf("%s is a %d-byte array — looks like a ROM image", path, typ.Len())
				}
				visit(typ.Elem(), path+"[]")
				return
			}
			if typ.Kind() != reflect.Struct {
				return
			}
			for i := 0; i < typ.NumField(); i++ {
				f := typ.Field(i)
				name := path + "." + f.Name
				// Skip third-party types (mos6502.Registers etc.).
				if !strings.HasPrefix(f.Type.PkgPath(), "github.com/gobeeb/GoBeeb/bbc") &&
					f.Type.PkgPath() != "" {
					continue
				}
				visit(f.Type, name)
			}
		}
		visit(root, root.Name())
	}
	walk(t, reflect.TypeOf(Snapshot{}))
}
