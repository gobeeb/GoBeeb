//go:build ignore

// gen.go fetches the SingleStepTests/ProcessorTests corpus into
// mos6502/testdata/processortests/ at a pinned upstream commit SHA via
// `git clone --depth 1` + sparse-checkout of the `6502/v1/` subtree.
//
// Invoked via `go generate ./mos6502/` (the //go:generate directive
// lives at the top of processortests_test.go). Build-tag `ignore` keeps
// this file out of normal package compilation.
//
// Contract: see specs/003-cpu-processor-tests/contracts/generator-contract.md.
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// pinnedCorpusSHA is the upstream commit this generator and the
// matching test harness were authored against. Mirrored in
// processortests_test.go for documentation parity; drift between the
// two declarations is caught in code review.
const pinnedCorpusSHA = "bb11756436da8fd16cce86aef63dc6725f48836f"

const upstreamRepoURL = "https://github.com/SingleStepTests/ProcessorTests.git"

// upstreamBranch is the ref this generator fetches. We pin reproducibility
// on `pinnedCorpusSHA` by verifying the resolved HEAD after checkout, not
// on the branch tip itself; if upstream advances past the pinned SHA the
// generator will fail loudly. A direct `git fetch --depth=1 origin <SHA>`
// avoids the branch indirection but, on this repo's history, fails on
// GitHub with "remote did not send all necessary objects" because the
// server insists on shipping parent commits for the SHA. The
// branch-tip+verify path sidesteps that protocol quirk while keeping the
// SHA the authoritative source of reproducibility.
const upstreamBranch = "main"

const corpusDir = "testdata/processortests"

const subtree = "6502/v1"

func main() {
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Fprintln(os.Stderr, "git not found on PATH: install git and retry")
		os.Exit(1)
	}

	if alreadyFetched() {
		fmt.Printf("processortests corpus already at %s; skipping fetch\n", pinnedCorpusSHA)
		return
	}

	if err := os.RemoveAll(corpusDir); err != nil {
		fmt.Fprintf(os.Stderr, "remove %s: %v\n", corpusDir, err)
		os.Exit(1)
	}
	if err := os.MkdirAll(corpusDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", corpusDir, err)
		os.Exit(1)
	}

	run("git", "init", corpusDir)
	runIn(corpusDir, "git", "remote", "add", "origin", upstreamRepoURL)
	runIn(corpusDir, "git", "sparse-checkout", "init", "--cone")
	runIn(corpusDir, "git", "sparse-checkout", "set", subtree)
	runIn(corpusDir, "git", "fetch", "--depth", "1", "origin", upstreamBranch)
	runIn(corpusDir, "git", "checkout", "FETCH_HEAD")

	// Reproducibility gate: the pinned SHA must equal the commit we
	// just checked out. If upstream has advanced past the pinned SHA,
	// the generator must fail loudly so the maintainer reconciles the
	// pin rather than silently fetching newer cases.
	head := captureIn(corpusDir, "git", "rev-parse", "HEAD")
	if head != pinnedCorpusSHA {
		fmt.Fprintf(os.Stderr, "fetched HEAD %s does not match pinnedCorpusSHA %s — upstream %s may have advanced; update pinnedCorpusSHA in mos6502/gen.go and mos6502/processortests_test.go after reviewing the diff\n",
			head, pinnedCorpusSHA, upstreamBranch)
		os.Exit(1)
	}

	marker := filepath.Join(corpusDir, ".fetched-sha")
	if err := os.WriteFile(marker, []byte(pinnedCorpusSHA+"\n"), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", marker, err)
		os.Exit(1)
	}

	fmt.Printf("processortests corpus fetched at %s\n", pinnedCorpusSHA)
}

func alreadyFetched() bool {
	data, err := os.ReadFile(filepath.Join(corpusDir, ".fetched-sha"))
	if err != nil {
		return false
	}
	if strings.TrimSpace(string(data)) != pinnedCorpusSHA {
		return false
	}
	matches, err := filepath.Glob(filepath.Join(corpusDir, subtree, "*.json"))
	if err != nil {
		return false
	}
	return len(matches) >= 200
}

func run(name string, args ...string) {
	runIn("", name, args...)
}

func runIn(dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s failed: %v\n", name, strings.Join(args, " "), err)
		if stderr.Len() > 0 {
			fmt.Fprintln(os.Stderr, stderr.String())
		}
		if name == "git" && len(args) > 0 && (args[0] == "fetch" || args[0] == "checkout") {
			fmt.Fprintf(os.Stderr, "hint: the pinned SHA %s may have been rewritten upstream; check pinnedCorpusSHA in mos6502/gen.go\n", pinnedCorpusSHA)
		}
		os.Exit(1)
	}
}

func captureIn(dir, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %s failed: %v\n", name, strings.Join(args, " "), err)
		os.Exit(1)
	}
	return strings.TrimSpace(string(out))
}
