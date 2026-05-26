# Contract — `go generate` Corpus Fetch

The corpus fetch tool. Lives at `mos6502/gen.go`, guarded by `//go:build ignore`. Invoked via a `//go:generate` directive at the top of `mos6502/processortests_test.go`:

```go
//go:generate go run gen.go
```

So `go generate ./mos6502/` runs `go run mos6502/gen.go` with `mos6502/` as the working directory.

---

## Inputs

| Input | Source | Required |
|---|---|---|
| Pinned upstream commit SHA | `const pinnedCorpusSHA` in `mos6502/gen.go` (mirrors the one in `processortests_test.go`) | yes |
| Upstream repo URL | `const upstreamRepoURL = "https://github.com/SingleStepTests/ProcessorTests.git"` in `mos6502/gen.go` | yes |
| Target directory | `mos6502/testdata/processortests/` (relative to working directory) | derived |
| `git` binary | system PATH | yes |
| Network access to `github.com` | runtime | yes |

---

## Outputs

| Output | Path (relative to repo root) | Owner |
|---|---|---|
| Corpus JSON files | `mos6502/testdata/processortests/6502/v1/*.json` | upstream content; generator only places |
| Idempotency marker | `mos6502/testdata/processortests/.fetched-sha` | generator |

Both outputs are gitignored via the new `.gitignore` rule (`mos6502/testdata/processortests/`).

---

## Behaviour contract

### Happy path — fresh

1. Verify `git` is on `PATH`. If not: `fmt.Fprintln(os.Stderr, "git not found on PATH"); os.Exit(1)`.
2. Verify `mos6502/testdata/processortests/.fetched-sha` either does not exist or does not equal `pinnedCorpusSHA`.
3. `os.RemoveAll("mos6502/testdata/processortests")`.
4. `git init mos6502/testdata/processortests` (creates the directory).
5. `cd mos6502/testdata/processortests && git remote add origin <upstreamRepoURL>`.
6. `git sparse-checkout init --cone`.
7. `git sparse-checkout set 6502/v1`.
8. `git fetch --depth 1 origin <pinnedCorpusSHA>`.
9. `git checkout FETCH_HEAD`.
10. Write `.fetched-sha` containing exactly `<pinnedCorpusSHA>\n`.
11. Print success: `fmt.Printf("processortests corpus fetched at %s\n", pinnedCorpusSHA)`.
12. `os.Exit(0)`.

### Happy path — already fetched (idempotency)

1. `mos6502/testdata/processortests/.fetched-sha` exists and contents == `pinnedCorpusSHA`.
2. `mos6502/testdata/processortests/6502/v1/` exists and contains > 200 `.json` files (sanity check; full count is 256 but allow some slack for corrupt manifests we haven't seen).
3. Print: `fmt.Printf("processortests corpus already at %s; skipping fetch\n", pinnedCorpusSHA)`.
4. `os.Exit(0)`.

### Partial-download recovery

If condition (1) or (2) of "already fetched" fails — for any reason: missing `.fetched-sha`, mismatched SHA, missing or thin `6502/v1/` — go to "fresh" path step 3 (RemoveAll + re-fetch). No clever recovery, no partial repair.

### Failure modes

| Condition | Behaviour |
|---|---|
| `git` not in PATH | exit 1, stderr message naming `git` as the missing dependency |
| `git fetch` fails (network, SHA unreachable) | exit 1, stderr message naming `pinnedCorpusSHA` as the unreachable target + the underlying `git` stderr |
| `git checkout` fails after a successful fetch | exit 1, stderr message; do NOT write `.fetched-sha` (leaves the directory in a partial state that the next run will detect via the missing-marker recovery path) |
| `os.RemoveAll` or `os.WriteFile` fails | exit 1, stderr message naming the path |

---

## Invariants

- Exit code 0 ⟺ on-disk state is `.fetched-sha == pinnedCorpusSHA` AND `6502/v1/` populated.
- Exit code non-zero ⟺ on-disk state is either untouched (pre-flight failure) or definitely-incomplete (`.fetched-sha` absent).
- The generator MUST NOT modify any file outside `mos6502/testdata/processortests/`.
- The generator MUST NOT depend on any Go runtime dependency outside the standard library.

---

## Non-goals

- Pretty progress bars / spinners. `git`'s own clone-progress output is sufficient.
- Resume of an interrupted clone. Wipe-and-retry is simpler and bounded (the corpus is small enough that re-fetching is fine).
- Multi-corpus support (`nes6502/v1`, `wdc65c02/v1`, etc.). Hard-coded to `6502/v1/` only. Future variants would touch this file.
- Network mirror fallback. If GitHub goes down, the harness can't fetch — escalate to a corpus mirror as a follow-up phase, not in scope here.
