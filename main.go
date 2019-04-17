package main

import (
	"github.com/gobeeb/GoBeeb/pkg/cpu"
)

const (
	RamInKB = 32
)

func main() {
	ram := make([]byte, RamInKB*1024)
	cpu := cpu.NewCpu(ram)
}
