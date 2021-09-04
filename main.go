package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gobeeb/GoBeeb/cpu"
)

const (
	RamInKB = 32
)

func main() {
	ram := make([]uint8, 65536)
	cpu := cpu.NewCPU(ram)

	if err := loadMemory("examples/6502_functional_test.bin", cpu); err != nil {
		panic(err)
	}

}

func loadMemory(path string, cpu *cpu.CPU) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	defer file.Close()

	totalRead := 0
	buf := make([]byte, 65536)

	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("reading file into memory: %w", err)
		}

		if n == 0 {
			break
		}

		totalRead++
	}

	var j uint16
	j = 0xc000
	for i, b := range buf {
		if i <= 15 {
			continue
		}
		cpu.Write(j, b)
		j++

		if j == 0xffff {
			break
		}
	}

	return nil
}
