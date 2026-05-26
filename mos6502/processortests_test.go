// Tom Harte SingleStepTests/ProcessorTests harness — per-cycle bus
// equivalence for every documented NMOS 6502 opcode against a real-chip
// oracle. See specs/003-cpu-processor-tests/ for the full design.
//
// Run modes:
//   - go test ./mos6502/           : full 10,000 cases per documented opcode
//   - go test -short ./mos6502/    : first 100 cases per documented opcode
//
// The corpus is fetched on demand:
//   - go generate ./mos6502/       : populates testdata/processortests/
//
//go:generate go run gen.go

package mos6502

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// pinnedCorpusSHA mirrors the constant of the same name in gen.go.
// Documented here so the test source is self-describing; drift between
// the two declarations is caught in code review and by
// TestPinnedSHADocumented (40 lowercase hex).
const pinnedCorpusSHA = "bb11756436da8fd16cce86aef63dc6725f48836f"

const corpusRoot = "testdata/processortests/6502/v1"

// shortModeCaseCap is the per-opcode case limit honoured under `go test
// -short`. Spec FR-007: 100 cases per documented opcode (~15,100 total).
const shortModeCaseCap = 100

// maxFailuresPerSubtest caps the number of per-case t.Errorf lines a
// failing subtest emits before bailing with t.Fatalf. Borrowed from the
// existing golden_trace_test.go style; the cap is required to keep
// Tom Harte failure noise bounded (R8).
const maxFailuresPerSubtest = 5

// ─────────────────────────────────────────────────────────────────────
// JSON case shape (E1)
// ─────────────────────────────────────────────────────────────────────

type processorCase struct {
	Name    string         `json:"name"`
	Initial processorState `json:"initial"`
	Final   processorState `json:"final"`
	Cycles  []busCycle     `json:"cycles"`
}

type processorState struct {
	PC  uint16     `json:"pc"`
	S   uint8      `json:"s"`
	A   uint8      `json:"a"`
	X   uint8      `json:"x"`
	Y   uint8      `json:"y"`
	P   uint8      `json:"p"`
	RAM []ramEntry `json:"ram"`
}

type ramEntry struct {
	Addr  uint16
	Value uint8
}

type busCycle struct {
	Addr  uint16
	Value uint8
	Kind  BusEventKind
}

func (s processorState) toRegisters() Registers {
	return Registers{
		A:      s.A,
		X:      s.X,
		Y:      s.Y,
		SP:     s.S,
		PC:     s.PC,
		P:      s.P,
		Cycles: 0,
	}
}

// UnmarshalJSON decodes upstream's [addr, value] 2-tuple shape.
func (r *ramEntry) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("ramEntry: %w", err)
	}
	if len(raw) != 2 {
		return fmt.Errorf("ramEntry: got %d elements, want 2", len(raw))
	}
	var addr, value int64
	if err := json.Unmarshal(raw[0], &addr); err != nil {
		return fmt.Errorf("ramEntry.addr: %w", err)
	}
	if err := json.Unmarshal(raw[1], &value); err != nil {
		return fmt.Errorf("ramEntry.value: %w", err)
	}
	if addr < 0 || addr > 0xFFFF {
		return fmt.Errorf("ramEntry: address %d out of range", addr)
	}
	if value < 0 || value > 0xFF {
		return fmt.Errorf("ramEntry: value %d out of range", value)
	}
	r.Addr = uint16(addr)
	r.Value = uint8(value)
	return nil
}

// UnmarshalJSON decodes upstream's [addr, value, "read"|"write"] 3-tuple.
//
//nolint:gocyclo // sequential validation of three positional JSON fields.
func (c *busCycle) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("busCycle: %w", err)
	}
	if len(raw) != 3 {
		return fmt.Errorf("busCycle: got %d elements, want 3", len(raw))
	}
	var addr, value int64
	if err := json.Unmarshal(raw[0], &addr); err != nil {
		return fmt.Errorf("busCycle.addr: %w", err)
	}
	if err := json.Unmarshal(raw[1], &value); err != nil {
		return fmt.Errorf("busCycle.value: %w", err)
	}
	var kindStr string
	if err := json.Unmarshal(raw[2], &kindStr); err != nil {
		return fmt.Errorf("busCycle.kind: %w", err)
	}
	if addr < 0 || addr > 0xFFFF {
		return fmt.Errorf("busCycle: address %d out of range", addr)
	}
	if value < 0 || value > 0xFF {
		return fmt.Errorf("busCycle: value %d out of range", value)
	}
	switch kindStr {
	case "read":
		c.Kind = BusRead
	case "write":
		c.Kind = BusWrite
	default:
		return fmt.Errorf("busCycle: unknown kind %q (want \"read\" or \"write\")", kindStr)
	}
	c.Addr = uint16(addr)
	c.Value = uint8(value)
	return nil
}

// ─────────────────────────────────────────────────────────────────────
// Sparse memory adapter (E2)
// ─────────────────────────────────────────────────────────────────────

type ramMap map[uint16]uint8

func (r ramMap) Read(addr uint16) uint8         { return r[addr] }
func (r ramMap) Write(addr uint16, value uint8) { r[addr] = value }

func newRAMMap(initial []ramEntry) ramMap {
	m := make(ramMap, len(initial)*2)
	for _, e := range initial {
		m[e.Addr] = e.Value
	}
	return m
}

// ─────────────────────────────────────────────────────────────────────
// Skip list (E3) — 105 undocumented NMOS opcodes
//
// Sources cross-checked:
//   - https://www.nesdev.org/wiki/CPU_unofficial_opcodes
//   - http://www.oxyron.de/html/opcodes02.html
//   - https://www.atarihq.com/danb/files/64doc.txt
//
// Mnemonic + addressing-mode tag (e.g., "SLO_zp", "LAX_indx") is
// recorded as the value so the future undocumented-opcode phase has a
// readable contract to remove entries one at a time.
// ─────────────────────────────────────────────────────────────────────

// Mnemonic constants for repeated entries; satisfies goconst without
// hiding the per-opcode literal at the call site.
const (
	mnKIL    = "KIL"
	mnDOPZp  = "DOP_zp"
	mnDOPZpX = "DOP_zpx"
	mnDOPImm = "DOP_imm"
	mnTOPAbx = "TOP_abx"
	mnNOPImp = "NOP_imp"
)

var skipList = map[uint8]string{
	// KILs (a.k.a. JAM / HLT) — 12 entries
	0x02: mnKIL, 0x12: mnKIL, 0x22: mnKIL, 0x32: mnKIL,
	0x42: mnKIL, 0x52: mnKIL, 0x62: mnKIL, 0x72: mnKIL,
	0x92: mnKIL, 0xB2: mnKIL, 0xD2: mnKIL, 0xF2: mnKIL,

	// NOP-family multi-byte/cycle illegals (DOP/TOP/NOP variants)
	0x04: mnDOPZp, 0x14: mnDOPZpX, 0x34: mnDOPZpX,
	0x44: mnDOPZp, 0x54: mnDOPZpX, 0x64: mnDOPZp,
	0x74: mnDOPZpX, 0x80: mnDOPImm, 0x82: mnDOPImm,
	0x89: mnDOPImm, 0xC2: mnDOPImm, 0xD4: mnDOPZpX,
	0xE2: mnDOPImm, 0xF4: mnDOPZpX,
	0x0C: "TOP_abs", 0x1C: mnTOPAbx, 0x3C: mnTOPAbx,
	0x5C: mnTOPAbx, 0x7C: mnTOPAbx, 0xDC: mnTOPAbx,
	0xFC: mnTOPAbx,
	0x1A: mnNOPImp, 0x3A: mnNOPImp, 0x5A: mnNOPImp,
	0x7A: mnNOPImp, 0xDA: mnNOPImp, 0xFA: mnNOPImp,

	// SLO (ASL + ORA) — 7 entries
	0x03: "SLO_indx", 0x07: "SLO_zp", 0x0F: "SLO_abs",
	0x13: "SLO_indy", 0x17: "SLO_zpx", 0x1B: "SLO_aby",
	0x1F: "SLO_abx",

	// RLA (ROL + AND) — 7 entries
	0x23: "RLA_indx", 0x27: "RLA_zp", 0x2F: "RLA_abs",
	0x33: "RLA_indy", 0x37: "RLA_zpx", 0x3B: "RLA_aby",
	0x3F: "RLA_abx",

	// SRE (LSR + EOR) — 7 entries
	0x43: "SRE_indx", 0x47: "SRE_zp", 0x4F: "SRE_abs",
	0x53: "SRE_indy", 0x57: "SRE_zpx", 0x5B: "SRE_aby",
	0x5F: "SRE_abx",

	// RRA (ROR + ADC) — 7 entries
	0x63: "RRA_indx", 0x67: "RRA_zp", 0x6F: "RRA_abs",
	0x73: "RRA_indy", 0x77: "RRA_zpx", 0x7B: "RRA_aby",
	0x7F: "RRA_abx",

	// SAX (A AND X store) — 4 entries
	0x83: "SAX_indx", 0x87: "SAX_zp", 0x8F: "SAX_abs",
	0x97: "SAX_zpy",

	// ANC / ALR / ARR / XAA / LAX-imm / AXS — immediate illegals
	0x0B: "ANC_imm", 0x2B: "ANC_imm",
	0x4B: "ALR_imm",
	0x6B: "ARR_imm",
	0x8B: "XAA_imm",
	0xAB: "LAX_imm",
	0xCB: "AXS_imm",
	0xEB: "SBC_imm_illegal",

	// "Magic" stores AHX / SHY / SHX / TAS / LAS
	0x93: "AHX_indy", 0x9F: "AHX_aby",
	0x9C: "SHY_abx",
	0x9E: "SHX_aby",
	0x9B: "TAS_aby",
	0xBB: "LAS_aby",

	// LAX (LDA + LDX) — 5 entries (LAX_imm 0xAB above)
	0xA3: "LAX_indx", 0xA7: "LAX_zp", 0xAF: "LAX_abs",
	0xB3: "LAX_indy", 0xB7: "LAX_zpy", 0xBF: "LAX_aby",

	// DCP (DEC + CMP) — 7 entries
	0xC3: "DCP_indx", 0xC7: "DCP_zp", 0xCF: "DCP_abs",
	0xD3: "DCP_indy", 0xD7: "DCP_zpx", 0xDB: "DCP_aby",
	0xDF: "DCP_abx",

	// ISC / ISB (INC + SBC) — 7 entries
	0xE3: "ISB_indx", 0xE7: "ISB_zp", 0xEF: "ISB_abs",
	0xF3: "ISB_indy", 0xF7: "ISB_zpx", 0xFB: "ISB_aby",
	0xFF: "ISB_abx",
}

// documentedOpcodes returns the set of opcode bytes that the
// production opcodeTable has a non-illegal entry for. The truth source
// is opcodeMetaTable.Illegal — set to false by opcodes.go for each of
// the 151 documented entries it installs over the default.
func documentedOpcodes() map[uint8]bool {
	out := make(map[uint8]bool, 151)
	for i := 0; i < 256; i++ {
		if !opcodeMetaTable[i].Illegal {
			out[uint8(i)] = true
		}
	}
	return out
}

// subtestName produces "0x<HH>_<MNEMONIC>_<addr_mode>" for documented
// opcodes, or "0x<HH>_<MNEMONIC>" (skipList value) for undocumented.
func subtestName(op uint8) string {
	if mn, ok := skipList[op]; ok {
		return fmt.Sprintf("0x%02X_%s", op, mn)
	}
	meta := opcodeMetaTable[op]
	suffix := modeShortName(meta.Mode)
	if suffix == "" {
		return fmt.Sprintf("0x%02X_%s", op, meta.Mnemonic)
	}
	return fmt.Sprintf("0x%02X_%s_%s", op, meta.Mnemonic, suffix)
}

//nolint:gocyclo // flat switch over the 13 addressing modes — splitting would only hide the table.
func modeShortName(m AddressingMode) string {
	switch m {
	case ModeImplicit:
		return "imp"
	case ModeAccumulator:
		return "acc"
	case ModeImmediate:
		return "imm"
	case ModeZeroPage:
		return "zp"
	case ModeZeroPageX:
		return "zpx"
	case ModeZeroPageY:
		return "zpy"
	case ModeRelative:
		return "rel"
	case ModeAbsolute:
		return "abs"
	case ModeAbsoluteX:
		return "abx"
	case ModeAbsoluteY:
		return "aby"
	case ModeIndirect:
		return "ind"
	case ModeIndexedIndirect:
		return "indx"
	case ModeIndirectIndexed:
		return "indy"
	}
	return ""
}

// ─────────────────────────────────────────────────────────────────────
// Per-case runner + failure formatter (FR-002, FR-008, R8)
// ─────────────────────────────────────────────────────────────────────

// runCase executes one Tom Harte case against a freshly-initialised CPU
// and reports any divergence on t. Returns true on diff so the caller
// can count failures against the per-subtest cap.
//
//nolint:gocyclo // three orthogonal assertion blocks (regs, ram, trace) by design.
func runCase(t *testing.T, c *processorCase, trace *Trace) bool {
	t.Helper()
	mem := newRAMMap(c.Initial.RAM)
	cpu := New(mem)
	cpu.SetRegisters(c.Initial.toRegisters())
	cpu.SetTrace(trace)
	trace.Reset()
	cpu.Step()

	want := c.Final.toRegisters()
	got := cpu.Registers()
	regOK := got.A == want.A && got.X == want.X && got.Y == want.Y &&
		got.SP == want.SP && got.PC == want.PC && got.P == want.P

	ramOK := true
	for _, e := range c.Final.RAM {
		if mem[e.Addr] != e.Value {
			ramOK = false
			break
		}
	}
	// Detect spurious writes: any address present in mem that's neither
	// in final.ram nor in initial.ram (or that diverges from its
	// initial value when final.ram has no entry for it).
	if ramOK {
		finalIdx := make(map[uint16]uint8, len(c.Final.RAM))
		for _, e := range c.Final.RAM {
			finalIdx[e.Addr] = e.Value
		}
		initialIdx := make(map[uint16]uint8, len(c.Initial.RAM))
		for _, e := range c.Initial.RAM {
			initialIdx[e.Addr] = e.Value
		}
		for addr, val := range mem {
			if _, inFinal := finalIdx[addr]; inFinal {
				continue
			}
			// Not listed in final → must equal initial value (or 0 if
			// not listed in initial either).
			if iv, inInitial := initialIdx[addr]; inInitial {
				if val != iv {
					ramOK = false
					break
				}
			} else if val != 0 {
				ramOK = false
				break
			}
		}
	}

	events := trace.Snapshot()
	traceOK := len(events) == len(c.Cycles)
	if traceOK {
		for i := range events {
			if events[i].Addr != c.Cycles[i].Addr ||
				events[i].Value != c.Cycles[i].Value ||
				events[i].Kind != c.Cycles[i].Kind {
				traceOK = false
				break
			}
		}
	}

	if regOK && ramOK && traceOK {
		return false
	}
	reportDiff(t, c, &got, &want, mem, events)
	return true
}

//nolint:gocyclo // three orthogonal diff blocks plus trace tail accounting — splitting would hide the failure shape.
func reportDiff(t *testing.T, c *processorCase, got, want *Registers, mem ramMap, events []BusEvent) {
	t.Helper()
	var b strings.Builder
	fmt.Fprintf(&b, "case %q:\n", c.Name)

	// Registers — print full block if any field differs.
	if got.A != want.A || got.X != want.X || got.Y != want.Y ||
		got.SP != want.SP || got.PC != want.PC || got.P != want.P {
		fmt.Fprintf(&b, "  registers:\n")
		fmt.Fprintf(&b, "    A : got=%02X want=%02X\n", got.A, want.A)
		fmt.Fprintf(&b, "    X : got=%02X want=%02X\n", got.X, want.X)
		fmt.Fprintf(&b, "    Y : got=%02X want=%02X\n", got.Y, want.Y)
		fmt.Fprintf(&b, "    SP: got=%02X want=%02X\n", got.SP, want.SP)
		fmt.Fprintf(&b, "    PC: got=%04X want=%04X\n", got.PC, want.PC)
		fmt.Fprintf(&b, "    P : got=%02X want=%02X\n", got.P, want.P)
	}

	// RAM — list only divergent cells.
	finalIdx := make(map[uint16]uint8, len(c.Final.RAM))
	for _, e := range c.Final.RAM {
		finalIdx[e.Addr] = e.Value
	}
	initialIdx := make(map[uint16]uint8, len(c.Initial.RAM))
	for _, e := range c.Initial.RAM {
		initialIdx[e.Addr] = e.Value
	}
	type ramDiff struct {
		addr          uint16
		got, want     uint8
		wantSpecified bool
		explainSpur   bool
	}
	var diffs []ramDiff
	for _, e := range c.Final.RAM {
		if mem[e.Addr] != e.Value {
			diffs = append(diffs, ramDiff{addr: e.Addr, got: mem[e.Addr], want: e.Value, wantSpecified: true})
		}
	}
	for addr, val := range mem {
		if _, ok := finalIdx[addr]; ok {
			continue
		}
		if iv, inInitial := initialIdx[addr]; inInitial {
			if val != iv {
				diffs = append(diffs, ramDiff{addr: addr, got: val, want: iv, wantSpecified: true, explainSpur: true})
			}
		} else if val != 0 {
			diffs = append(diffs, ramDiff{addr: addr, got: val, want: 0, explainSpur: true})
		}
	}
	if len(diffs) > 0 {
		sort.Slice(diffs, func(i, j int) bool { return diffs[i].addr < diffs[j].addr })
		fmt.Fprintf(&b, "  ram diff (%d cell(s)):\n", len(diffs))
		for _, d := range diffs {
			note := ""
			if d.explainSpur {
				note = " (spurious write)"
			}
			fmt.Fprintf(&b, "    $%04X: got=%02X want=%02X%s\n", d.addr, d.got, d.want, note)
		}
	}

	// Trace — first divergent cycle index.
	if len(events) != len(c.Cycles) {
		fmt.Fprintf(&b, "  cycles: got len=%d want len=%d\n", len(events), len(c.Cycles))
	}
	minLen := len(events)
	if len(c.Cycles) < minLen {
		minLen = len(c.Cycles)
	}
	for i := 0; i < minLen; i++ {
		if events[i].Addr != c.Cycles[i].Addr ||
			events[i].Value != c.Cycles[i].Value ||
			events[i].Kind != c.Cycles[i].Kind {
			fmt.Fprintf(&b, "  first divergent cycle [%d]: got=%s want=%s\n",
				i, fmtBusEvent(events[i].Addr, events[i].Value, events[i].Kind),
				fmtBusEvent(c.Cycles[i].Addr, c.Cycles[i].Value, c.Cycles[i].Kind))
			break
		}
	}
	if len(events) > len(c.Cycles) {
		i := len(c.Cycles)
		fmt.Fprintf(&b, "  extra observed cycle [%d]: got=%s\n", i,
			fmtBusEvent(events[i].Addr, events[i].Value, events[i].Kind))
	} else if len(events) < len(c.Cycles) {
		i := len(events)
		fmt.Fprintf(&b, "  missing expected cycle [%d]: want=%s\n", i,
			fmtBusEvent(c.Cycles[i].Addr, c.Cycles[i].Value, c.Cycles[i].Kind))
	}

	t.Errorf("%s", b.String())
}

func fmtBusEvent(addr uint16, value uint8, kind BusEventKind) string {
	k := "read"
	if kind == BusWrite {
		k = "write"
	}
	return fmt.Sprintf("{addr=$%04X value=%02X kind=%s}", addr, value, k)
}

// ─────────────────────────────────────────────────────────────────────
// Driver (FR-003, FR-007, FR-012)
// ─────────────────────────────────────────────────────────────────────

var corpusCheckOnce sync.Once
var corpusCheckErr error

func ensureCorpus(t *testing.T) {
	t.Helper()
	corpusCheckOnce.Do(func() {
		if _, err := os.Stat(corpusRoot); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				corpusCheckErr = fmt.Errorf("corpus not found at mos6502/%s/; run `go generate ./mos6502/` to fetch", corpusRoot)
				return
			}
			corpusCheckErr = fmt.Errorf("stat %s: %w", corpusRoot, err)
		}
	})
	if corpusCheckErr != nil {
		t.Fatal(corpusCheckErr)
	}
}

func TestProcessorTests(t *testing.T) {
	ensureCorpus(t)

	for op := 0; op < 256; op++ {
		op := uint8(op)
		name := subtestName(op)
		if mn, skip := skipList[op]; skip {
			t.Run(name, func(t *testing.T) {
				t.Skipf("undocumented: %s (Phase 003 OOS-001)", mn)
			})
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runOpcodeSubtest(t, op)
		})
	}
}

func runOpcodeSubtest(t *testing.T, op uint8) {
	path := filepath.Join(corpusRoot, fmt.Sprintf("%02x.json", op))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var cases []processorCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}

	limit := len(cases)
	if testing.Short() && limit > shortModeCaseCap {
		limit = shortModeCaseCap
	}

	trace := NewTrace(16)
	failures := 0
	for i := 0; i < limit; i++ {
		if runCase(t, &cases[i], trace) {
			failures++
			if failures >= maxFailuresPerSubtest {
				t.Fatalf("opcode 0x%02X: %d failures (cap reached); skipping remaining %d cases", op, failures, limit-i-1)
			}
		}
	}
}

// ─────────────────────────────────────────────────────────────────────
// Harness-validation tests (tests-of-tests)
// ─────────────────────────────────────────────────────────────────────

func TestPinnedSHADocumented(t *testing.T) {
	if got := len(pinnedCorpusSHA); got != 40 {
		t.Fatalf("pinnedCorpusSHA length = %d, want 40", got)
	}
	for i, r := range pinnedCorpusSHA {
		ok := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !ok {
			t.Fatalf("pinnedCorpusSHA[%d] = %q, want lowercase hex", i, r)
		}
	}
}

func TestSkipListInvariants(t *testing.T) {
	if got := len(skipList); got != 105 {
		t.Fatalf("len(skipList) = %d, want 105", got)
	}
	doc := documentedOpcodes()
	if got := len(doc); got != 151 {
		t.Fatalf("documentedOpcodes count = %d, want 151", got)
	}
	for op, mn := range skipList {
		if mn == "" {
			t.Errorf("skipList[0x%02X] = \"\" — every value must be non-empty", op)
		}
		if doc[op] {
			t.Errorf("skipList[0x%02X] (%s) overlaps with documented set", op, mn)
		}
	}
	// documented + skipped MUST = 256, no overlap.
	for op := 0; op < 256; op++ {
		_, skipped := skipList[uint8(op)]
		documented := doc[uint8(op)]
		if skipped == documented {
			t.Errorf("opcode 0x%02X: documented=%v skipped=%v (must be exactly one)", op, documented, skipped)
		}
	}
}

func TestRAMMapRoundTrip(t *testing.T) {
	initial := []ramEntry{{Addr: 0x1234, Value: 0xAA}, {Addr: 0x5678, Value: 0xBB}}
	m := newRAMMap(initial)
	if got := m.Read(0x1234); got != 0xAA {
		t.Errorf("Read($1234) = %02X, want AA", got)
	}
	if got := m.Read(0x5678); got != 0xBB {
		t.Errorf("Read($5678) = %02X, want BB", got)
	}
	if got := m.Read(0x9ABC); got != 0x00 {
		t.Errorf("Read($9ABC) = %02X, want 00 (unmapped → zero)", got)
	}
	m.Write(0x9ABC, 0xCC)
	if got := m.Read(0x9ABC); got != 0xCC {
		t.Errorf("Read($9ABC) after Write = %02X, want CC", got)
	}

	// Final-state comparison: every final entry present, every unrelated key unchanged.
	final := []ramEntry{{Addr: 0x1234, Value: 0xAA}, {Addr: 0x5678, Value: 0xDD}, {Addr: 0x9ABC, Value: 0xCC}}
	m.Write(0x5678, 0xDD)
	for _, e := range final {
		if m[e.Addr] != e.Value {
			t.Errorf("final $%04X: got %02X want %02X", e.Addr, m[e.Addr], e.Value)
		}
	}
}

func TestRAMEntryUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantA   uint16
		wantV   uint8
	}{
		{"positive 2-tuple", `[35714, 169]`, false, 35714, 169},
		{"zero values", `[0, 0]`, false, 0, 0},
		{"max values", `[65535, 255]`, false, 0xFFFF, 0xFF},
		{"malformed object", `{"addr": 1}`, true, 0, 0},
		{"address overflow", `[65536, 0]`, true, 0, 0},
		{"value overflow", `[0, 256]`, true, 0, 0},
		{"negative address", `[-1, 0]`, true, 0, 0},
		{"too few elements", `[1]`, true, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var r ramEntry
			err := r.UnmarshalJSON([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Fatalf("UnmarshalJSON(%q): err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
			if err == nil {
				if r.Addr != tc.wantA || r.Value != tc.wantV {
					t.Errorf("UnmarshalJSON(%q): got {%04X,%02X} want {%04X,%02X}",
						tc.input, r.Addr, r.Value, tc.wantA, tc.wantV)
				}
			}
		})
	}
}

func TestBusCycleUnmarshal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		wantA   uint16
		wantV   uint8
		wantK   BusEventKind
	}{
		{"read tuple", `[35714, 169, "read"]`, false, 35714, 169, BusRead},
		{"write tuple", `[35714, 169, "write"]`, false, 35714, 169, BusWrite},
		{"unknown kind", `[0, 0, "fetch"]`, true, 0, 0, 0},
		{"case-sensitive kind", `[0, 0, "READ"]`, true, 0, 0, 0},
		{"malformed object", `{"addr": 1}`, true, 0, 0, 0},
		{"address overflow", `[65536, 0, "read"]`, true, 0, 0, 0},
		{"value overflow", `[0, 256, "read"]`, true, 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var c busCycle
			err := c.UnmarshalJSON([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Fatalf("UnmarshalJSON(%q): err=%v wantErr=%v", tc.input, err, tc.wantErr)
			}
			if err == nil {
				if c.Addr != tc.wantA || c.Value != tc.wantV || c.Kind != tc.wantK {
					t.Errorf("UnmarshalJSON(%q): got {%04X,%02X,%v} want {%04X,%02X,%v}",
						tc.input, c.Addr, c.Value, c.Kind, tc.wantA, tc.wantV, tc.wantK)
				}
			}
		})
	}
}

// TestRunCaseDetectsDivergence is the SC-006 self-check: runCase MUST
// report a failure when register, RAM, or trace state diverges from the
// case's expected `final` / `cycles`. Without this, a silently-broken
// harness could pass on every documented opcode by accident.
//
// Each subtest takes the same baseline case (LDA #$AA at $0200, expected
// final A=$AA, two read cycles) and perturbs exactly one piece of the
// expected output before invoking runCase. A correct harness reports
// failure for each perturbation; a broken harness would erroneously pass.
func TestRunCaseDetectsDivergence(t *testing.T) {
	baseline := func() processorCase {
		return processorCase{
			Name: "synthetic LDA #$AA",
			Initial: processorState{
				PC: 0x0200, S: 0xFD, A: 0x00, X: 0x00, Y: 0x00, P: 0x24,
				RAM: []ramEntry{{Addr: 0x0200, Value: 0xA9}, {Addr: 0x0201, Value: 0xAA}},
			},
			Final: processorState{
				PC: 0x0202, S: 0xFD, A: 0xAA, X: 0x00, Y: 0x00, P: 0xA4,
				RAM: []ramEntry{{Addr: 0x0200, Value: 0xA9}, {Addr: 0x0201, Value: 0xAA}},
			},
			Cycles: []busCycle{
				{Addr: 0x0200, Value: 0xA9, Kind: BusRead},
				{Addr: 0x0201, Value: 0xAA, Kind: BusRead},
			},
		}
	}

	// Sanity: baseline must PASS without perturbation.
	t.Run("baseline_passes", func(t *testing.T) {
		c := baseline()
		fakeT := &testing.T{}
		if runCase(fakeT, &c, NewTrace(16)) {
			t.Fatalf("baseline runCase returned divergence; harness oracle is wrong")
		}
	})

	perturbations := []struct {
		name    string
		mutate  func(c *processorCase)
		message string
	}{
		{
			name:    "register diff",
			mutate:  func(c *processorCase) { c.Final.A = 0x55 },
			message: "wrong final A should be reported",
		},
		{
			name:    "trace cycle count diff",
			mutate:  func(c *processorCase) { c.Cycles = append(c.Cycles, busCycle{Addr: 0x0202, Value: 0, Kind: BusRead}) },
			message: "extra expected cycle should be reported",
		},
		{
			name:    "trace cycle order diff",
			mutate:  func(c *processorCase) { c.Cycles[0].Addr = 0x0FFF },
			message: "wrong first-cycle address should be reported",
		},
		{
			name:    "trace cycle kind diff",
			mutate:  func(c *processorCase) { c.Cycles[1].Kind = BusWrite },
			message: "wrong cycle kind should be reported",
		},
		{
			name:    "ram diff",
			mutate:  func(c *processorCase) { c.Final.RAM = append(c.Final.RAM, ramEntry{Addr: 0x0300, Value: 0xFF}) },
			message: "missing RAM cell should be reported",
		},
	}
	for _, p := range perturbations {
		t.Run(p.name, func(t *testing.T) {
			c := baseline()
			p.mutate(&c)
			fakeT := &testing.T{}
			if !runCase(fakeT, &c, NewTrace(16)) {
				t.Fatalf("%s: runCase returned PASS on perturbed case (harness blind to divergence)", p.message)
			}
		})
	}
}
