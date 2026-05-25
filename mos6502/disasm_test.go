package mos6502

import "testing"

func TestDisassemble(t *testing.T) {
	cases := []struct {
		bytes      []byte
		pc         uint16
		wantText   string
		wantLength int
	}{
		{[]byte{0xA9, 0x42}, 0x0600, "LDA #$42", 2},
		{[]byte{0xA5, 0x10}, 0x0600, "LDA $10", 2},
		{[]byte{0xB5, 0x10}, 0x0600, "LDA $10,X", 2},
		{[]byte{0xAD, 0x34, 0x12}, 0x0600, "LDA $1234", 3},
		{[]byte{0xBD, 0x34, 0x12}, 0x0600, "LDA $1234,X", 3},
		{[]byte{0xB9, 0x34, 0x12}, 0x0600, "LDA $1234,Y", 3},
		{[]byte{0xA1, 0x10}, 0x0600, "LDA ($10,X)", 2},
		{[]byte{0xB1, 0x10}, 0x0600, "LDA ($10),Y", 2},
		{[]byte{0x4C, 0x00, 0x06}, 0x0600, "JMP $0600", 3},
		{[]byte{0x6C, 0xFF, 0x10}, 0x0600, "JMP ($10FF)", 3},
		{[]byte{0x90, 0x10}, 0x0600, "BCC $0612", 2}, // 0x0600 + 2 + 0x10 = 0x0612
		{[]byte{0x90, 0xFE}, 0x0600, "BCC $0600", 2}, // 0x0600 + 2 + (-2) = 0x0600
		{[]byte{0x0A}, 0x0600, "ASL A", 1},
		{[]byte{0xEA}, 0x0600, "NOP", 1},
		{[]byte{0x60}, 0x0600, "RTS", 1},
		{[]byte{0x00}, 0x0600, "BRK", 1},
		{[]byte{0xB6, 0x10}, 0x0600, "LDX $10,Y", 2},
	}
	for _, c := range cases {
		t.Run(c.wantText, func(t *testing.T) {
			var ram flatRAM
			copy(ram[c.pc:], c.bytes)
			text, length := Disassemble(&ram, c.pc)
			if text != c.wantText {
				t.Errorf("text=%q want %q", text, c.wantText)
			}
			if length != c.wantLength {
				t.Errorf("length=%d want %d", length, c.wantLength)
			}
		})
	}
}
