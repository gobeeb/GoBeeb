package bbc

import (
	"bytes"
	_ "embed"
	"errors"
	"testing"
)

//go:embed testdata/stub_os_16k.bin
var stubOSROM []byte

//go:embed testdata/stub_sideways_aa.bin
var stubSidewaysAA []byte

//go:embed testdata/stub_sideways_55.bin
var stubSideways55 []byte

func TestLoadOSROM_AcceptsExactly16KiB(t *testing.T) {
	m := New()
	if err := m.LoadOSROM(stubOSROM); err != nil {
		t.Fatalf("LoadOSROM(16 KiB stub): %v", err)
	}
	if !m.rom.osLoaded {
		t.Fatal("osLoaded should be true after LoadOSROM")
	}
}

func TestLoadOSROM_RejectsWrongSize(t *testing.T) {
	cases := []struct {
		name string
		size int
	}{
		{"empty", 0},
		{"one short", romImageSize - 1},
		{"one long", romImageSize + 1},
		{"half", romImageSize / 2},
		{"double", romImageSize * 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := New()
			err := m.LoadOSROM(make([]byte, tc.size))
			if !errors.Is(err, ErrInvalidROMSize) {
				t.Fatalf("got %v, want ErrInvalidROMSize", err)
			}
			if m.rom.osLoaded {
				t.Fatal("osLoaded must remain false on rejected load")
			}
		})
	}
}

func TestLoadOSROM_CopyOnLoad(t *testing.T) {
	m := New()
	src := make([]byte, romImageSize)
	copy(src, stubOSROM)
	if err := m.LoadOSROM(src); err != nil {
		t.Fatalf("LoadOSROM: %v", err)
	}
	// Mutate caller's slice; machine must be unaffected.
	for i := range src {
		src[i] = 0xAA
	}
	if !bytes.Equal(m.rom.os[:], stubOSROM) {
		t.Fatal("machine's OS ROM mutated when caller mutated source slice")
	}
}

func TestLoadSidewaysROM_AcceptsExactly16KiB(t *testing.T) {
	m := New()
	if err := m.LoadSidewaysROM(0, stubSidewaysAA); err != nil {
		t.Fatalf("LoadSidewaysROM(0, AA): %v", err)
	}
	if !m.rom.sidewaysLoaded[0] {
		t.Fatal("sidewaysLoaded[0] should be true")
	}
	if m.rom.sideways[0][0] != 0xAA {
		t.Fatalf("sideways[0][0]=%#x, want $AA", m.rom.sideways[0][0])
	}
}

func TestLoadSidewaysROM_RejectsWrongSize(t *testing.T) {
	m := New()
	err := m.LoadSidewaysROM(0, make([]byte, romImageSize-1))
	if !errors.Is(err, ErrInvalidROMSize) {
		t.Fatalf("got %v, want ErrInvalidROMSize", err)
	}
	if m.rom.sidewaysLoaded[0] {
		t.Fatal("sidewaysLoaded[0] must remain false on rejected load")
	}
}

func TestLoadSidewaysROM_RejectsBankOutOfRange(t *testing.T) {
	m := New()
	for _, bank := range []int{-1, 4, 5, 100} {
		err := m.LoadSidewaysROM(bank, make([]byte, romImageSize))
		if !errors.Is(err, ErrBankOutOfRange) {
			t.Fatalf("bank=%d: got %v, want ErrBankOutOfRange", bank, err)
		}
	}
}

func TestLoadSidewaysROM_CopyOnLoad(t *testing.T) {
	m := New()
	src := make([]byte, romImageSize)
	copy(src, stubSidewaysAA)
	if err := m.LoadSidewaysROM(2, src); err != nil {
		t.Fatalf("LoadSidewaysROM: %v", err)
	}
	for i := range src {
		src[i] = 0x00
	}
	if !bytes.Equal(m.rom.sideways[2][:], stubSidewaysAA) {
		t.Fatal("machine's sideways bank mutated when caller mutated source slice")
	}
}
