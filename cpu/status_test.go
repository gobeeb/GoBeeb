package cpu_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/gobeeb/GoBeeb/cpu"
)

func TestStatus(t *testing.T) {
	RegisterTestingT(t)

	var s cpu.Status

	s.Set(cpu.D)
	s.Toggle(cpu.C)

	Expect(s.Has(cpu.D)).To(BeTrue())
	Expect(s.Has(cpu.C)).To(BeTrue())

}
