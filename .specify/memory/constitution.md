<!--
Sync Impact Report
==================
Version change: (none) → 1.0.0
Modified principles: N/A (initial ratification)
Added sections:
  - Core Principles
    - I. Code Quality
    - II. Testing Standards (NON-NEGOTIABLE)
    - III. User Experience Consistency
    - IV. Performance Requirements
  - Quality Gates & Standards
  - Development Workflow
  - Governance
Removed sections: N/A
Templates requiring updates:
  - ✅ .specify/templates/plan-template.md (Constitution Check gate is generic — resolves against this file at plan time; no rewrite needed)
  - ✅ .specify/templates/spec-template.md (no constitution references)
  - ✅ .specify/templates/tasks-template.md (no constitution references)
  - ✅ .specify/templates/checklist-template.md (no constitution references)
Follow-up TODOs:
  - TODO(PROJECT_VISION): Replace placeholder project description in the Preamble once a product brief or README lands.
-->

# GoBeeb Constitution

## Preamble

GoBeeb is a software project under active development. This constitution defines the
non-negotiable principles, quality gates, and governance rules that all contributors,
agents, and automated workflows MUST honor. It is the source of truth for "how we
build here" and supersedes informal practice when the two conflict.

TODO(PROJECT_VISION): Replace this preamble with the canonical product description
once a README or PRD is authored.

## Core Principles

### I. Code Quality

Code MUST be readable, reviewable, and maintainable by someone other than its author.

- All code MUST pass the project's automated formatter and linter before merge; CI
  enforces this as a hard gate.
- Functions MUST have a single, named responsibility. Cyclomatic complexity SHOULD
  stay at or below 10; deviations require an inline justification comment.
- Public APIs (exported functions, types, endpoints) MUST carry doc comments stating
  purpose, inputs, outputs, and error modes.
- Dead code, commented-out code, and TODOs without an owner or tracking link MUST NOT
  be merged.
- Code review MUST explicitly verify naming, structure, error handling, and absence
  of duplication. A bare "LGTM" with no substantive comments is not a review.

**Rationale**: Most defects and onboarding friction trace to unreadable or
over-complicated code. Enforcing quality at the gate is cheaper than remediating it
after the fact.

### II. Testing Standards (NON-NEGOTIABLE)

Tests prove behavior; they are not optional documentation.

- Every new feature MUST ship with unit tests covering its happy path and at least
  one failure mode. Every bug fix MUST ship with a regression test that fails without
  the fix and passes with it.
- Delta line coverage on new or changed code MUST be ≥ 80%. Overall project coverage
  MUST NOT regress between merges.
- Integration tests MUST cover external service boundaries, persistence layers, and
  any public API contract. Contract changes require contract test updates in the
  same PR.
- Tests MUST be deterministic. Flaky tests MUST be quarantined within one business
  day and fixed or deleted within five business days; quarantine is not a parking
  lot.
- The test pyramid is the default shape: many fast unit tests, fewer integration
  tests, a small set of E2E tests. Inverted pyramids require explicit justification
  in the plan's Complexity Tracking section.

**Rationale**: Untested code is liability disguised as velocity. Strict, deterministic
testing is what lets us refactor and ship confidently.

### III. User Experience Consistency

The product MUST feel like one product, not a collage of independent features.

- All user-facing surfaces MUST consume the shared design tokens (color, spacing,
  typography, motion). One-off styles MUST be promoted into the shared system or
  removed.
- Interaction patterns — loading, empty, error, success, confirmation, destructive
  action — MUST follow the shared component library. New patterns require a design
  review before first use.
- User-visible copy MUST follow the project voice guide. Error messages MUST be
  actionable: state what happened, why, and the user's next step.
- Accessibility floor: WCAG 2.1 Level AA. Keyboard navigation, focus order, color
  contrast, and semantic markup MUST be verified before merge for any UI change.
- Cross-platform parity: a feature that exists on more than one surface MUST behave
  identically unless a platform constraint forbids it, in which case the divergence
  MUST be documented.

**Rationale**: Inconsistency taxes users with cognitive load and erodes trust.
Treating consistency as a first-class constraint, not a polish item, keeps the
surface coherent as it grows.

### IV. Performance Requirements

Performance is a feature with explicit, enforced budgets — not a "we'll optimize
later" afterthought.

- User-facing interactive operations MUST meet: p95 < 200 ms, p99 < 500 ms, measured
  at the user-perceptible boundary (not just service internals).
- Every new endpoint, job, or batch operation MUST declare its latency and throughput
  budget in the plan before implementation. Budgets are reviewed alongside the
  design.
- Bundle size, binary size, and memory footprint MUST be tracked per build. A
  regression of >5% on any tracked budget blocks merge until justified in writing.
- Algorithms that process more than 1,000 items per request, or run on a hot path,
  MUST be profiled and have a stated worst-case complexity in a code comment.
- Performance regressions discovered post-release MUST be triaged within one business
  day and have a rollback or fix plan within three.

**Rationale**: Performance debt compounds silently. Declaring budgets up-front and
enforcing them automatically is the only reliable way to keep the product fast as it
scales.

## Quality Gates & Standards

The following gates run in CI and MUST pass before any merge to `master`:

1. **Format & lint**: project formatter and linter, zero warnings on changed lines.
2. **Unit & integration tests**: all tests pass; delta coverage ≥ 80%.
3. **Security scan**: dependency vulnerability scan and secret scan; no new HIGH or
   CRITICAL findings.
4. **Performance budget check**: bundle/binary size and benchmark deltas within
   declared budgets.
5. **Accessibility check** (UI changes only): automated a11y scan with zero new
   violations at WCAG 2.1 AA.
6. **Constitution Check**: plans produced via `/speckit-plan` MUST explicitly state
   compliance with each of the four Core Principles, or justify deviation in the
   Complexity Tracking section.

Gate failures MUST be fixed at the root cause. Bypassing a gate (`--no-verify`,
disabling a CI check, lowering a threshold) requires maintainer approval, an issue
documenting the exception, and a deadline for re-enabling the gate.

## Development Workflow

- **Branching**: trunk-based development on `master`. Feature branches MUST be
  short-lived (≤ 5 days) and rebased onto `master` before merge.
- **Reviews**: every PR requires at least one reviewer who is not the author.
  Changes touching the constitution, CI configuration, or shared infrastructure
  require maintainer approval.
- **Specification flow**: non-trivial work (new feature, behavioral change,
  cross-cutting refactor) MUST follow the Spec-Kit flow:
  `/speckit-constitution` → `/speckit-specify` → `/speckit-plan` → `/speckit-tasks`
  → `/speckit-implement`. Trivial work (typo, doc tweak, dependency bump) may
  bypass this flow with a one-line PR description.
- **Commits**: atomic, with descriptive messages. The "why" belongs in the commit
  body, not just the diff.
- **Issue hygiene**: every merged PR MUST link to an issue or spec. Drive-by
  changes unrelated to the stated scope MUST be split into separate PRs.

## Governance

- **Authority**: this constitution supersedes ad-hoc practices, individual
  preferences, and prior conventions. When practice and constitution conflict,
  constitution wins until amended.
- **Amendments**: amendments MUST be proposed via PR that (a) edits this file,
  (b) states the rationale, (c) bumps the version per the rules below,
  (d) propagates changes to any dependent templates under `.specify/templates/`,
  and (e) is approved by a maintainer.
- **Versioning policy** (semantic):
  - **MAJOR**: a principle is removed, or its meaning is redefined in a
    backward-incompatible way; governance authority is restructured.
  - **MINOR**: a new principle or section is added, or an existing principle is
    materially expanded.
  - **PATCH**: clarifications, wording fixes, typo corrections, or non-semantic
    refinements.
- **Compliance review**: maintainers MUST review constitution compliance at least
  quarterly. Violations MUST be logged as issues and remediated on a stated
  timeline; unremediated violations escalate to an amendment proposal (either fix
  the practice or fix the constitution).
- **Runtime guidance**: agent-facing guidance lives in `CLAUDE.md` (and
  equivalents for other agents). Those files MUST defer to this constitution on
  any conflict.

**Version**: 1.0.0 | **Ratified**: 2026-05-25 | **Last Amended**: 2026-05-25
