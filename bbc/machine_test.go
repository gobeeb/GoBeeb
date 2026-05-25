package bbc

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReset_NoOSROM(t *testing.T) {
	m := New()
	if err := m.Reset(); !errors.Is(err, ErrNoOSROM) {
		t.Fatalf("Reset() with no OS ROM: got %v, want ErrNoOSROM", err)
	}
	if err := m.ColdReset(); !errors.Is(err, ErrNoOSROM) {
		t.Fatalf("ColdReset() with no OS ROM: got %v, want ErrNoOSROM", err)
	}
}

func TestReset_PreservesRAM(t *testing.T) {
	m := newWithStubOS(t)
	// Seed RAM via direct map access (same goroutine, same package).
	m.mmap.ram[0x100] = 0x42
	m.mmap.ram[0x7FFF] = 0xA5
	m.rom.bank = 3

	if err := m.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if m.mmap.ram[0x100] != 0x42 || m.mmap.ram[0x7FFF] != 0xA5 {
		t.Fatal("Reset must preserve RAM")
	}
	if m.rom.bank != 0 {
		t.Fatalf("Reset must clear bank latch, got %d", m.rom.bank)
	}
}

func TestColdReset_ZerosRAM(t *testing.T) {
	m := newWithStubOS(t)
	for i := range m.mmap.ram {
		m.mmap.ram[i] = 0xCD
	}
	m.rom.bank = 2

	if err := m.ColdReset(); err != nil {
		t.Fatalf("ColdReset: %v", err)
	}
	for i, b := range m.mmap.ram {
		if b != 0 {
			t.Fatalf("ColdReset must zero RAM; ram[%#x]=%#x", i, b)
		}
	}
	if m.rom.bank != 0 {
		t.Fatalf("ColdReset must clear bank latch, got %d", m.rom.bank)
	}
}

// FR-021: idle stubs never pull IRQ/NMI low during a long Tick run
// without any peripheral interaction.
func TestTick_NoSpontaneousInterrupts(t *testing.T) {
	m := newWithStubOS(t)
	if err := m.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	// Run > 10 000 cycles. If a stub were spontaneously pulling
	// IRQ low, the CPU would service it and PC would drift into
	// the IRQ vector at $C000 (NOPs there are fine, but the
	// architectural P.I flag would flip). We assert the I flag
	// never moves away from its post-RESET set-state and that
	// the cumulative cycle count keeps advancing.
	initial := m.cpu.Registers().Cycles
	after := m.Tick(10_000)
	if after-initial < 10_000 {
		t.Fatalf("Tick advanced only %d cycles, want >= 10000", after-initial)
	}
	regs := m.cpu.Registers()
	const flagI = 0x04
	if regs.P&flagI == 0 {
		t.Fatal("I flag cleared during idle Tick — interrupt was serviced")
	}
}

// FR-028: no internal locks or atomics on the hot path. Reflection
// over Machine and MemoryMap asserts the policy structurally.
func TestNoLocksOrChannelsOnHotPath(t *testing.T) {
	forbid := []string{"sync.Mutex", "sync.RWMutex", "atomic."}
	check := func(t *testing.T, typ reflect.Type) {
		t.Helper()
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			name := f.Type.String()
			if f.Type.Kind() == reflect.Chan {
				t.Errorf("%s.%s is a channel; forbidden on Machine/MemoryMap", typ.Name(), f.Name)
			}
			for _, bad := range forbid {
				if strings.Contains(name, bad) {
					t.Errorf("%s.%s has forbidden type %q", typ.Name(), f.Name, name)
				}
			}
		}
	}
	check(t, reflect.TypeOf(Machine{}))
	check(t, reflect.TypeOf(MemoryMap{}))
}

// newWithStubOS builds a Machine and loads the embedded stub OS
// ROM. Common helper for tests that need a ready-to-Tick machine.
func newWithStubOS(t *testing.T) *Machine {
	t.Helper()
	m := New()
	if err := m.LoadOSROM(stubOSROM); err != nil {
		t.Fatalf("LoadOSROM(stub): %v", err)
	}
	return m
}
