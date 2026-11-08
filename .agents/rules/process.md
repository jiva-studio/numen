# Development Process & Roles

This rule defines the roles, the TDD lifecycle, and what an agent must prove before handing work over.

The pipeline: `/nm-feature` (owner and agent write the feature file) then `/nm-specify` (local spec) then `tester` (red tests) then `coder` (green) then `/nm-review`. The scenarios in `features/` are the lasting acceptance criteria and follow [`gherkin.md`](./gherkin.md). `spec.md` and `scope.md` in `.agents/tasks/<slug>/` are local files and are not committed.

---

## 1. Roles

### Owner

- Is the only author of the scenarios in `features/`. Approves each feature file.

### Test Agent (`tester`)

- Writes the steps and the driver under `tests/acceptance/desktop/` for the scenarios, and the lower-level tests (Go, vitest, stories) the scenario map of `spec.md` lists.
- **Writes tests red first** and proves they are red: the failing test run output must be demonstrated. A test that has never failed has not been shown to test anything.
- **Owns those files**: the steps, the driver and the lower-level tests are its output, and nobody else edits them. It does not edit the scenarios.
- **Pure behavioral naming**: test names describe observable behavior (e.g. `it('rejects card with empty front')`). Never use synthetic tags or prefixes (`AC-1`, `(D-1)`).

### Implementation Agent (`coder`)

- Takes red tests to green according to `spec.md`.
- **Never edits scenarios or acceptance files.** If a scenario looks wrong, or the feature grew beyond it, the coder proposes the change to the feature file in a commit of its own whose message says so, and lists it in its final report. If a test looks wrong, it reports it and waits. Editing tests to match the implementation destroys the independent verification check.
- Writes narrow unit tests for internal helpers and unexported functions. Those are its own files.

### Adversarial Reviewer (`reviewer`)

- **Mandate: break it.** Not to confirm it works — to find where it does not, where it goes beyond the scenarios, the spec or `scope.md`, or breaks architectural boundaries.
- **Obligation: prove it.** A finding is a failing experiment (a test, a command, a trace), not a vague suspicion.
- **No right to fix.** The reviewer reports findings. The coder repairs them.
- **Always lists every scenario added, changed or removed** in the diff.

---

## 2. What is frozen first

**The contract — domain types, port interfaces, wire protocol, and migrations — is frozen before implementation starts.** It is the shape every other part of the codebase codes against.

Once frozen:

- It is not edited without human consultation.
- Fixtures move with types: a type changed without its fixtures leaves the suite compiling against one shape and testing against another.

---

## 3. What an agent must prove before handover

Before handing work over, an agent must demonstrate concrete evidence:

1. **The gate is green**: The relevant test suites and linters pass (`go test ./...`, `npm run typecheck`, `npm run test:unit`, `npm run format:check`).
2. **Acceptance tests are verified**: The scenarios' tests were proven red before implementation and are green after.
3. **No regressions in existing suites**: Existing tests continue to pass.

---

## 4. Where a test belongs

Scenarios in `features/` cover business rules. Technical end-to-end tests (`tests/e2e/desktop`) stay for what needs the whole app but is not a business rule. Checks of layout or file formats belong to stories, vitest or Go tests.

---

## 5. Decisions reserved for the human

An agent does not decide these alone. It prepares the change, explains the tradeoffs, and asks the human:

- **Permissions and security boundaries** — anything touching authorization or encryption keys.
- **Database migrations** — altering existing tables, indexes, or schema constraints.
- **The wire contract** — changes to protocol buffers or shared API payloads between client and server.
- **Dependencies** — adding, removing, or upgrading external third-party libraries.
