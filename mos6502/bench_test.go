package mos6502

import "testing"

// BenchmarkRunNoop is a tight NOP loop. NOP is 2 cycles. We loop via
// a long sequence of NOPs followed by a JMP back to the start.
func BenchmarkRunNoop(b *testing.B) {
	var ram flatRAM
	// 256 NOPs then JMP $0600.
	for i := 0; i < 256; i++ {
		ram[0x0600+i] = 0xEA
	}
	ram[0x0700] = 0x4C
	ram[0x0701] = 0x00
	ram[0x0702] = 0x06

	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})

	b.ReportAllocs()
	b.ResetTimer()
	startCycles := cpu.cycles
	for i := 0; i < b.N; i++ {
		cpu.Step()
	}
	cyclesRun := cpu.cycles - startCycles
	b.ReportMetric(float64(cyclesRun), "cycles")
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(cyclesRun), "ns/cycle")
}

// BenchmarkRunMixedWorkload runs a representative mix exercising
// loads, stores, branches, RMW, and decimal arithmetic.
func BenchmarkRunMixedWorkload(b *testing.B) {
	var ram flatRAM
	// Program at $0600:
	//  LDA #$05
	//  STA $80
	//  INC $80
	//  LDX $80
	//  DEX
	//  BNE -7      ; branch back to INC
	//  LDA #$10
	//  CLC
	//  ADC #$05
	//  JMP $0600
	prog := []byte{
		0xA9, 0x05, // LDA #$05
		0x85, 0x80, // STA $80
		0xE6, 0x80, // INC $80
		0xA6, 0x80, // LDX $80
		0xCA,       // DEX
		0xD0, 0xF9, // BNE -7
		0xA9, 0x10, // LDA #$10
		0x18,       // CLC
		0x69, 0x05, // ADC #$05
		0x4C, 0x00, 0x06, // JMP $0600
	}
	copy(ram[0x0600:], prog)

	cpu := New(&ram)
	cpu.SetRegisters(Registers{PC: 0x0600, SP: 0xFD, P: FlagUnused})

	b.ReportAllocs()
	b.ResetTimer()
	startCycles := cpu.cycles
	for i := 0; i < b.N; i++ {
		cpu.Step()
	}
	cyclesRun := cpu.cycles - startCycles
	if cyclesRun > 0 {
		b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(cyclesRun), "ns/cycle")
	}
}
