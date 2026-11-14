---
name: bug-hunter
description: Stage 2 of the numen review pipeline. Conducts a deep semantic audit of the diff and its blast radius for logic bugs, races, reactivity breaks, unhandled edges and behavioural regressions. Reports findings; does not change files.
---

# Stage 2 — Semantic Logic & Blast Radius

A white-box audit looking only for **functional defects**: logic bugs, races, corrupted state, regressions.

---

## Core principles

1. **Zero style noise.** Ignore formatting, naming, placement and file size entirely — Stage 1 and the linters cover those. Every finding here describes a failure at runtime.
2. **Adversarial reading.** Do not read the code assuming it works. Assume the author made a subtle false assumption about state, ordering, or an edge value.
3. **Proof of bug.** No "this could theoretically fail". Every finding includes the inputs and the state transitions that lead to the corruption or the crash.

---

## Phase 1: Diff and blast radius

1. Extract the change for the target resolved in [`../SKILL.md`](../SKILL.md): `git diff "$BASE_SHA...$HEAD_SHA"` for a pull request or commit range, `git diff HEAD` for an uncommitted working tree.
2. Map the blast radius:
   - Every caller of a changed function, method or component.
   - Upstream sources of the state it reads: props, RPC responses, index rows, files on disk.
   - Downstream consumers: watchers, computed properties, event subscribers, renderers, background workers.

A change to a wire message affects both the Go side and the generated TypeScript client. A change to what the index stores affects every query over it and needs a numbered migration.

---

## Phase 2: Defect matrix

### 1. Go — concurrency and lifetime

- Data races on shared maps, slices or structs without a mutex or an atomic; a guard covering most accesses rather than every one.
- A goroutine started without an owner that knows when it ends, or one that never selects on `ctx.Done()`.
- A blocking channel send with no buffer and no `select` default.
- `time.Sleep` used as synchronisation, in code or in a test.
- One process, one lifetime: work outliving the thing that started it.

### 2. Go — context, errors and resources

- `ctx` not first, named otherwise, or stored in a struct; not propagated to something that blocks; cancellation observed only between loop iterations and not inside one.
- An error wrapped without `%w`, or wrapped with nothing that locates the failure.
- An error both logged and returned, or dropped in an empty branch.
- A sentinel compared by string instead of `errors.Is`; a type not reached with `errors.As`.
- `panic` for bad input or a missing file.
- `defer` missing on something opened; `rows.Err()` unchecked; `defer tx.Rollback()` without the commit last; `Close` unchecked where the write matters.
- SQL built by string formatting where a parameter would do; a stored path that is absolute or platform-shaped instead of relative and slashed; a write that must not tear going straight to the target instead of through a temporary file and a rename.
- Time read from deep inside instead of injected where it changes behaviour.

### 3. Vue — reactivity and lifecycle

- Destructuring a reactive object or a store without `toRefs()` / `storeToRefs()`; mutating a prop; mutating a `readonly()` structure.
- A `watch` doing what a `computed` should, so derived state goes stale; a watcher missing `flush: 'post'` when it reads updated DOM, or `immediate: true` when the initial state needs handling.
- A closure capturing outdated state in an async callback or a listener.
- A listener, timer, observer or subscription attached with no unmount that ends it.
- A promise resolving after unmount and mutating state or navigating.

### 4. Async ordering

- Two actions fired in quick succession where the first resolves last and overwrites the second's result — missing cancellation or sequencing.
- A stream or subscription whose events can arrive interleaved with a refetch.
- Optimistic state written before the call it depends on has succeeded, leaving the interface inconsistent when the call rejects.

### 5. Boundaries and indexes

- Off-by-one in a range, a slice, a page or a cursor.
- Bytes against UTF-16 code units: a _stretch_ is bytes over the source's text, a _span_ is `from`/`to` as a client counts it, and nothing crosses that boundary unconverted.
- A string sliced by byte where a rune or a grapheme was meant.
- Empty input, a single element, a trailing newline, a file with no final newline, CRLF against LF.

### 6. Wire contracts and data integrity

- A field optional on the Go side (`omitempty`, a nullable column) read on the TypeScript side with no guard.
- A generated type used against a hand-written shape that has drifted.
- A value renamed on the way across the protocol boundary.

### 7. A failure stays a failure

- Follow every error, cancellation and missing value to the place that decides what to do with it. No path on the way may turn it into something the caller takes for success: a default value, a partial result, or a fallback to other data than the caller asked for. It is worst when a write follows a failed read.
- What the caller gave is what the work uses, and what the work opens it closes, on every path including an exception.

### 8. Conditions

- Compound conditions inverted: `!a && b` where `!(a && b)` was meant.
- A `switch` with no default where a new case is plausible; a fallthrough nobody intended.

---

## Output integration

Every defect becomes a finding in the report in [`../SKILL.md`](../SKILL.md), in the format given there, ordered by severity rather than by position in the file. Do not output a standalone report.

Pass every unproven hypothesis to Stage 3.
