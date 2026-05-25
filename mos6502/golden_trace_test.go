package mos6502

import (
	"fmt"
	"strings"
	"testing"
)

// goldenCase describes one micro-program and its expected bus trace.
type goldenCase struct {
	name    string
	program []byte // loaded at $0600
	setup   func(c *CPU, ram *flatRAM)
	want    string // canonical multi-line trace
}

// formatTrace renders a Trace's snapshot to a canonical multi-line
// form: `cycle: R|W $addr = $value` per event.
func formatTrace(tr *Trace) string {
	var b strings.Builder
	for _, e := range tr.Snapshot() {
		kind := "R"
		if e.Kind == BusWrite {
			kind = "W"
		}
		fmt.Fprintf(&b, "%d: %s $%04X = $%02X\n", e.Cycle, kind, e.Addr, e.Value)
	}
	return b.String()
}

// TestGoldenTraces asserts the per-cycle bus pattern for a hand-picked
// set of opcode/addressing-mode combinations that exercise the NMOS
// quirks: page-cross dummy reads, NMOS JMP-indirect page bug, RMW
// double-write, accumulator-form (no memory cycles for operand).
// (US4 acceptance scenario 1, SC-008)
func TestGoldenTraces(t *testing.T) {
	cases := []goldenCase{
		{
			name:    "lda_imm",
			program: []byte{0xA9, 0x42},
			want: `1: R $0600 = $A9
2: R $0601 = $42
`,
		},
		{
			name:    "lda_abs",
			program: []byte{0xAD, 0x34, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				ram[0x1234] = 0x77
			},
			want: `1: R $0600 = $AD
2: R $0601 = $34
3: R $0602 = $12
4: R $1234 = $77
`,
		},
		{
			name:    "lda_abs_x_nopagecross",
			program: []byte{0xBD, 0x10, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				c.X = 0x05
				ram[0x1215] = 0x55
			},
			want: `1: R $0600 = $BD
2: R $0601 = $10
3: R $0602 = $12
4: R $1215 = $55
`,
		},
		{
			name:    "lda_abs_x_pagecross",
			program: []byte{0xBD, 0xFF, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				c.X = 0x01
				ram[0x1300] = 0x99
			},
			// On page-cross we read the un-fixed-up address ($1200) as a dummy
			// then the real one ($1300).
			want: `1: R $0600 = $BD
2: R $0601 = $FF
3: R $0602 = $12
4: R $1200 = $00
5: R $1300 = $99
`,
		},
		{
			name:    "sta_abs_x_always_penalty",
			program: []byte{0x9D, 0x10, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				c.A = 0xAA
				c.X = 0x05
			},
			// Store always pays the dummy-read cycle even without page-cross.
			want: `1: R $0600 = $9D
2: R $0601 = $10
3: R $0602 = $12
4: R $1215 = $00
5: W $1215 = $AA
`,
		},
		{
			name:    "jmp_indirect_normal",
			program: []byte{0x6C, 0x34, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				ram[0x1234] = 0x00
				ram[0x1235] = 0x08
			},
			want: `1: R $0600 = $6C
2: R $0601 = $34
3: R $0602 = $12
4: R $1234 = $00
5: R $1235 = $08
`,
		},
		{
			name:    "jmp_indirect_pagebug",
			program: []byte{0x6C, 0xFF, 0x10},
			setup: func(c *CPU, ram *flatRAM) {
				ram[0x10FF] = 0x34 // target low
				ram[0x1000] = 0x12 // NMOS bug: high read from $1000, not $1100
			},
			want: `1: R $0600 = $6C
2: R $0601 = $FF
3: R $0602 = $10
4: R $10FF = $34
5: R $1000 = $12
`,
		},
		{
			name:    "inc_abs_rmw_double_write",
			program: []byte{0xEE, 0x34, 0x12},
			setup: func(c *CPU, ram *flatRAM) {
				ram[0x1234] = 0x41
			},
			want: `1: R $0600 = $EE
2: R $0601 = $34
3: R $0602 = $12
4: R $1234 = $41
5: W $1234 = $41
6: W $1234 = $42
`,
		},
		{
			name:    "asl_accumulator_no_memory_cycles",
			program: []byte{0x0A},
			setup: func(c *CPU, ram *flatRAM) {
				c.A = 0x42
			},
			// 2 cycles total: opcode fetch + dummy fetch at PC.
			want: `1: R $0600 = $0A
2: R $0601 = $00
`,
		},
		{
			name:    "brk_normal",
			program: []byte{0x00, 0x00},
			setup: func(c *CPU, ram *flatRAM) {
				ram[0xFFFE] = 0x00
				ram[0xFFFF] = 0x80
			},
			// BRK: opcode + padding + push PCH + push PCL + push P + vector lo + vector hi
			want: `1: R $0600 = $00
2: R $0601 = $00
3: W $01FD = $06
4: W $01FC = $02
5: W $01FB = $30
6: R $FFFE = $00
7: R $FFFF = $80
`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cpu, ram := newTestCPU(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})
			copy(ram[0x0600:], c.program)
			if c.setup != nil {
				c.setup(cpu, ram)
			}
			tr := NewTrace(64)
			cpu.SetTrace(tr)
			cpu.Step()
			got := formatTrace(tr)
			if got != c.want {
				t.Errorf("trace mismatch:\n--- got ---\n%s--- want ---\n%s", got, c.want)
			}
		})
	}
}
