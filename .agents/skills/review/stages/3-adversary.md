---
name: adversary
description: Stage 3 of the numen review pipeline. Writes and runs adversarial tests (go test / vitest) to prove or disprove the hypotheses from Stage 2, and checks that the change's own tests can fail.
---

# Stage 3 — Dynamic Stress Verification

The tester's mindset: **if the code can fail under some sequence of events, write the test that forces it to fail.**

---

## Core principles

1. **Code over opinion.** A hypothesis is not a defect until a test execution fails against the current implementation.
2. **Zero false positives.** A finding promoted from Stage 2 to "confirmed" carries the failing run.
3. **A test is an asset.** Every adversarial test that uncovered something stays in the suite as a regression test once the fix lands. One that proved resilience stays if it covers an edge nobody had covered.

This stage writes tests. It does not fix the implementation.

---

## The mutation check, first

Before attacking, turn the check on the change's own tests. For anything guarding a rule rather than a value, the question is not "does it pass?" but **"would it fail if the rule were broken?"** — and the way to know is to break it and watch.

Remove the filter, invert the condition, delete the guard clause, run the test, put it back. This is required for tests defending what the type system cannot: vault scoping, invalidation, anything about what is *not* read or *not* returned. It matters as much on the interface side, where "the nodes were arranged" and "the component rendered" pass under almost any implementation.

A test that passes either way is worse than no test, because it is counted as protection and nobody looks at it again. Report it as a Stage 3 finding with the mutation that did not break it.

---

## Attack vectors

### 1. Zero, empty, enormous

Empty vault, empty note, a note with no frontmatter, broken YAML, a file that is not markdown, a hidden folder. A single character. A source of ten thousand chunks. A line with nothing to break at. Text that is not Latin, combining diacritics, RTL, emoji past the basic plane. CRLF against LF, and a missing final newline.

The committed fixture vault in `tests/` is awkward on purpose and holds most of these. It is read-only: a test that needs to change it works on a copy.

### 2. Two vaults, disjoint content

Any claim about scoping is tested with two vaults whose notes share no words, asserting both directions: the first does not return the second's notes, and the second does not return the first's. Two vaults pointing at one folder prove nothing, because a leak then looks like a plausible number.

### 3. Concurrency and ordering

- `go test -race` with several connections held at once, since anything configured per connection is handed the same connection by a sequential test and cannot observe that the others were never configured.
- A cancelled context part-way through a scan, a walk or a query: does the work stop, and does it leave the index consistent?
- In Vitest, resolve mocked responses out of order — the first request sent, the second sent, the first resolving last.
- Rapid-fire interaction: repeated clicks, keyboard navigation before an animation or a fetch settles, a double submit, a double toggle.
- Unmount while an operation is in flight; assert no unhandled rejection and no warning.

### 4. Invariants

- A file written and read back is the same bytes.
- An index rebuilt from the same vault gives the same rows.
- A chunk identified by its text is the same identifier on either platform (*A chunk is identified by its text*).
- The pure core of a component gives the same numbers on any machine on any day: run it twice with the same props and a fixed clock.

---

## Execution protocol

1. **Formulate.** From the diff and Stage 2's hypotheses, name one to three scenarios intended to break an invariant.
2. **Craft.** An adjacent test file: `[name]_test.go` beside the Go code, `[name].spec.ts` beside the TypeScript. Minimal fixtures, the boundary or the race injected, the assertion explicit. `t.TempDir`, `t.Context`, `t.Cleanup` — never the machine's real home, config or cache directory, and never the network.
3. **Run.**
   ```bash
   cd modules/libs/core && go test -race -run TestName ./path/...
   cd modules/apps/desktop/editor && npx vitest run --project unit <path>.spec.ts
   ```
   For a story, render it in both engines: `vitest run --project 'stories (chromium)'` and `vitest run --project 'stories (webkit)'`, one at a time.
4. **Evaluate.**
   - The test **fails** → the defect is proven. Report the trace and the minimal reproduction, and mark the Stage 2 card confirmed.
   - The test **passes** → the code is resilient against that vector. Say so.

---

## Output integration

Every result goes into **Stage 3** of the Unified Report in [`../SKILL.md`](../SKILL.md), in the table format given there. Name any test file left behind in the Action Items. Do not output a standalone report.
