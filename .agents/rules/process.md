# Development Process & Roles

This rule defines the roles, the TDD lifecycle, and what an agent must prove before handing work over.

---

## 1. Roles

### Test Agent (`tester`)
- Derives acceptance criteria from `spec.md` and turns each one into an executable test.
- **Writes tests red first** and proves they are red: the failing test run output must be demonstrated. A test that has never failed has not been shown to test anything.
- **Owns acceptance test files**: they are its output, and nobody else edits them.
- **Pure behavioral naming**: test names describe observable behavior (e.g. `it('rejects card with empty front')`). Never use synthetic tags or prefixes (`AC-1`, `(D-1)`).

### Implementation Agent (`coder`)
- Takes red tests to green according to `spec.md`.
- **Does not touch acceptance test files**. If a test looks wrong — wrong expectation, wrong fixture, wrong criterion — report it and wait. Editing tests to match the implementation destroys the independent verification check.
- Writes narrow unit tests for internal helpers and unexported functions. Those are its own files; acceptance tests are not.

### Adversarial Reviewer (`reviewer`)
- **Mandate: break it.** Not to confirm it works — to find where it does not, where it violates Non-Goals in `intent.md`, or breaks architectural boundaries.
- **Obligation: prove it.** A finding is a failing experiment (a test, a command, a trace), not a vague suspicion.
- **No right to fix.** The reviewer reports findings. The coder repairs them.

---

## 2. What is frozen first

**The contract — domain types, port interfaces, wire protocol, and migrations — is frozen before implementation starts.** It is the shape every other part of the codebase codes against.

Once frozen:
- It is not edited without human consultation.
- Fixtures move with types: a type changed without its fixtures leaves the suite compiling against one shape and testing against another.

---

## 3. What an agent must prove before handover

Before handing work over, an agent must demonstrate concrete evidence:

1. **The gate is green**: The relevant test suites and linters pass (`go test ./...`, `vue-tsc`, `vitest`).
2. **Acceptance tests are verified**: Acceptance tests were proven red before implementation and are green after.
3. **No regressions in existing suites**: Existing tests continue to pass.

---

## 4. What stays with the human

An agent does not decide these alone. It prepares the change, explains the tradeoffs, and asks the human:

- **Permissions and security boundaries** — anything touching authorization or encryption keys.
- **Database migrations** — altering existing tables, indexes, or schema constraints.
- **The wire contract** — changes to protocol buffers or shared API payloads between client and server.
- **Dependencies** — adding, removing, or upgrading external third-party libraries.
