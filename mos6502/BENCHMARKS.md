# mos6502 Benchmark Record

Performance baseline established for regression comparison.
Target: ≤ 125 ns per emulated CPU cycle on amd64 (SC-006).
Zero allocations per emulated cycle (`-benchmem` 0 B/op, 0 allocs/op).

## 2026-05-25 — v1.0 baseline

Host: AMD Ryzen Threadripper PRO 3945WX (24 cores), Linux x86-64.
Toolchain: `go test -bench=. -benchmem -benchtime=3s`.

| Benchmark                  | ns/op | ns/cycle | allocs |
|----------------------------|------:|---------:|-------:|
| BenchmarkRunNoop           |  8.51 |   4.25   |   0    |
| BenchmarkRunMixedWorkload  | 14.08 |   4.34   |   0    |

Both benchmarks are ~30× faster than the SC-006 budget. Headroom to
spare for future feature additions (interrupts at higher granularity,
RDY sub-instruction stretching, etc.).

## Regression policy

A >5% increase in either ns/cycle figure blocks merge until justified.
Re-run with `make bench` and update this table on intentional
regressions, with a one-sentence note explaining the cause.
