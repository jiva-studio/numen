# Development Process & Roles

This rule defines the roles, the TDD lifecycle, and what an agent must prove before handing work over.

The pipeline: `/nm-feature` (owner and agent write the feature file) then `/nm-specify` (local spec) then `tester` (red tests) then `coder` (green) then `/nm-review`. The scenarios in `features/` are the lasting acceptance criteria and follow [`gherkin.md`](./gherkin.md). `spec.md` and `scope.md` in .agents/tasks/<slug>/ are local files and are not committed.

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

## 4. Pull request pipeline

Every pull request follows this sequence:

1. **Open as a draft**: Push the branch and create a draft pull request (`gh pr create --draft`).
2. **Bring CI to green**: Wait for all CI checks on the draft pull request to pass (`gh pr checks --watch`). Fix any failure and push to the draft branch until CI is green. A draft skips the end-to-end tests, the flashcards tests and the acceptance scenarios.
3. **Mark ready; the review starts after green CI**: Once CI is green, mark the pull request ready for review (`gh pr ready`). CI runs again, now with the tests and the scenarios tagged `@smoke` in Chromium (see [`gherkin.md`](./gherkin.md), Tags). When that run is green, `.github/workflows/claude-review.yml` runs the nm-review skill on the pull request and submits its verdict as an approval or a change request from github-actions[bot]. Only the first green run starts a review: a pull request that already has a review from github-actions[bot] waits for `/nm-review`. Set the Status of the issue to In review (section 7). Wait for CI (`gh pr checks --watch`), then for the review run (`gh run watch` on the latest `claude-review.yml` run), and read the verdict (`gh pr view <N> --json reviews`). Do not run the nm-review skill locally for this step: the approval from the workflow is the one the branch rules count.
4. **Fix and review again**: Fix every finding, push, wait for CI (each push to a ready pull request runs the `@smoke` tests in Chromium), and comment `/nm-review` on the pull request (`gh pr comment <N> --body "/nm-review"`; `/nm-review full` also writes and runs the tests that prove its hypotheses) until the verdict is an approval. A push to main runs all the selected end-to-end tests and acceptance scenarios in Chromium and WebKit.
5. **Request CodeRabbit and merge**: Once CI is green and the review workflow has approved, trigger CodeRabbit (`gh pr comment <N> --body "@coderabbitai review"`). The pull request is squash-merged without waiting for CodeRabbit; findings it posts before the merge are fixed.

---

## 5. Where a test belongs

Scenarios in `features/` cover business rules. Technical end-to-end tests (`tests/e2e/desktop`) stay for what needs the whole app but is not a business rule. Checks of layout or file formats belong to stories, vitest or Go tests.

A mutation survivor is killed by a new or stronger test. For `budget.go:42 CONDITIONALS_BOUNDARY: a > b → a >= b`, a test calls the function with `a == b` and asserts the result, and the pull request `test(flashcards-studio): kill the budget survivors` changes `budget_test.go` only. A survivor that no test can kill is an equivalent mutant or dead code: the equivalent mutant gets an inline directive with its reason, and the dead code is removed in a separate `refactor` pull request that leaves the test pull request as it is. Production code does not change in a `test` pull request; CI fails one that does (`tools/git-hooks/check-test-files.mjs`, the directive forms in `tools/ci-mutation/README.md`).

---

## 6. Decisions reserved for the human

An agent does not decide these alone. It prepares the change, explains the tradeoffs, and asks the human:

- **Permissions and security boundaries** — anything touching authorization or encryption keys.
- **Database migrations** — altering existing tables, indexes, or schema constraints.
- **The wire contract** — changes to protocol buffers or shared API payloads between client and server.
- **Dependencies** — adding, removing, or upgrading external third-party libraries.

---

## 7. Tracking: the project, the status and the milestone

Work is tracked in the project [Numen](https://github.com/orgs/jiva-studio/projects/15) of the organization. Every issue is in it; a pull request is not, it is seen on its issue.

| Field     | Where       | Values                                      | Who sets it                                                              |
| --------- | ----------- | ------------------------------------------- | ------------------------------------------------------------------------ |
| Type      | the issue   | Feature, Bug, Task, Tech debt               | whoever files the issue                                                  |
| Priority  | the issue   | Urgent, High, Medium, Low                   | the owner                                                                |
| Effort    | the issue   | Low, Medium, High                           | the owner                                                                |
| Area      | the project | core, plugins, ui, ci, mobile, docs         | whoever files the issue                                                  |
| Status    | the project | Backlog, Todo, In Progress, In review, Done | below                                                                    |
| Milestone | the issue   | `YYYY.MM`, the release of that month        | the owner for the plan; the milestone workflow for a merged pull request |

A milestone is named after the month of its release and has the last day of the month as its due date. An open issue in a milestone is planned for that release; a closed one shipped in it. A milestone is closed when its release is published.

An agent keeps the status of its issue true:

1. **Takes an issue**: sets Status to In Progress before the first commit. An agent works only on an issue whose Status is Todo or In Progress; an issue in Backlog is the owner's to move.
2. **Marks the pull request ready** (step 3 of the pipeline): sets Status to In review.
3. **Merge or close**: Done, set by the built-in project workflows (a closed issue and a merged pull request set it). An agent does not set Done by hand.
4. **Files a new issue**: sets its Type and Priority, adds it to the project with its Area, and leaves Status at Backlog. The labels of an issue only name what the three fields do not.

The commands:

```
gh project item-add 15 --owner jiva-studio --url <issue url>
gh project item-edit --id <item id> --project-id PVT_kwDOBtvOi84Brz7D --field-id <Status field id> --single-select-option-id <option id>
```

The ids come from `gh project field-list 15 --owner jiva-studio --format json` and `gh project item-list 15 --owner jiva-studio --format json`.

A pull request is written with `Closes #N`, so the issue and the pull request move together. The workflow `milestone.yml` puts a merged pull request, the issues it closes and an issue closed as completed in the milestone of the month when they have none.
