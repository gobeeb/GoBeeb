package bbc

import (
	"testing"
)

type unmappedEvent struct {
	addr  uint16
	write bool
	value uint8
}

func captureUnmapped(m *Machine) *[]unmappedEvent {
	events := make([]unmappedEvent, 0, 16)
	m.SetUnmappedAccessHook(func(addr uint16, write bool, value uint8) {
		events = append(events, unmappedEvent{addr, write, value})
	})
	return &events
}

func TestUnmappedHook_FiresOnFREDReadsAndWrites(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	if got := m.mmap.Read(0xFC00); got != 0xFF {
		t.Fatalf("$FC00 read returned %#x, want $FF", got)
	}
	m.mmap.Write(0xFC42, 0x77)

	if len(*events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(*events), *events)
	}
	if (*events)[0] != (unmappedEvent{0xFC00, false, 0xFF}) {
		t.Fatalf("read event: %+v", (*events)[0])
	}
	if (*events)[1] != (unmappedEvent{0xFC42, true, 0x77}) {
		t.Fatalf("write event: %+v", (*events)[1])
	}
}

func TestUnmappedHook_FiresOnJIMReadsAndWrites(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	_ = m.mmap.Read(0xFD00)
	m.mmap.Write(0xFDFF, 0x11)

	if len(*events) != 2 {
		t.Fatalf("got %d events, want 2", len(*events))
	}
	if (*events)[0].addr != 0xFD00 || (*events)[0].write {
		t.Fatalf("first event: %+v", (*events)[0])
	}
	if (*events)[1].addr != 0xFDFF || !(*events)[1].write || (*events)[1].value != 0x11 {
		t.Fatalf("second event: %+v", (*events)[1])
	}
}

func TestUnmappedHook_FiresOnSHEILAGap(t *testing.T) {
	// $FE38–$FE3F is the documented gap between the ROM-select
	// latch ($FE30–$FE33) + ACCCON ($FE34–$FE37) and the System
	// VIA ($FE40+). It must always be unmapped.
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	_ = m.mmap.Read(0xFE38)
	m.mmap.Write(0xFE3F, 0x5A)

	if len(*events) != 2 {
		t.Fatalf("got %d events, want 2: %+v", len(*events), *events)
	}
	if (*events)[0] != (unmappedEvent{0xFE38, false, 0xFF}) {
		t.Fatalf("read event: %+v", (*events)[0])
	}
	if (*events)[1] != (unmappedEvent{0xFE3F, true, 0x5A}) {
		t.Fatalf("write event: %+v", (*events)[1])
	}
}

func TestUnmappedHook_SilentOnRAM(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	m.mmap.Write(0x1234, 0x42)
	if got := m.mmap.Read(0x1234); got != 0x42 {
		t.Fatalf("RAM round-trip: got %#x, want $42", got)
	}
	if len(*events) != 0 {
		t.Fatalf("RAM accesses must not fire hook; got %+v", *events)
	}
}

func TestUnmappedHook_SilentOnOSROM(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	_ = m.mmap.Read(0xC000)
	_ = m.mmap.Read(0xFFFC) // vector region
	m.mmap.Write(0xC100, 0xAB)
	m.mmap.Write(0xFFFF, 0xCD)

	if len(*events) != 0 {
		t.Fatalf("OS ROM accesses must not fire hook; got %+v", *events)
	}
}

func TestUnmappedHook_NilIsSilent(t *testing.T) {
	m := newWithStubOS(t)
	// No SetUnmappedAccessHook call — hook is nil.
	if got := m.mmap.Read(0xFC00); got != 0xFF {
		t.Fatalf("$FC00 read with nil hook returned %#x, want $FF", got)
	}
	m.mmap.Write(0xFC00, 0x99)
	// Reaching this line without a panic is the assertion.
}

func TestUnmappedHook_ValueCarriesOpenBusAndWriteByte(t *testing.T) {
	m := newWithStubOS(t)
	events := captureUnmapped(m)

	_ = m.mmap.Read(0xFE38)
	m.mmap.Write(0xFE38, 0xC3)

	if (*events)[0].value != 0xFF {
		t.Fatalf("read event value=%#x, want $FF (open-bus)", (*events)[0].value)
	}
	if (*events)[1].value != 0xC3 {
		t.Fatalf("write event value=%#x, want $C3 (attempted byte)", (*events)[1].value)
	}
}
