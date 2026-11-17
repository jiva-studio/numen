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
4. **A failure is fixed, not repeated**: a flaky test is never rerun until it passes and a timeout is never raised; the cause is found and fixed. A check is never weakened and gets no exclusion list.

Heavy runs (the Go and vitest suites, builds, `make checks`, mutation runs) go to the build host, not to the laptop; the laptop runs formatters, git and gh.

---

## 4. Pull request pipeline

Every pull request follows this sequence:

1. **Open as a draft**: Push the branch and create a draft pull request (`gh pr create --draft`).
2. **Bring CI to green**: Wait for all CI checks on the draft pull request to pass (`gh pr checks --watch`). Fix any failure and push to the draft branch until CI is green. A draft skips the end-to-end tests, the flashcards tests and the acceptance scenarios.
3. **Mark ready; the review starts after green CI**: Once CI is green, mark the pull request ready for review (`gh pr ready`). CI runs again, now with the tests and the scenarios tagged `@smoke` in Chromium (see [`gherkin.md`](./gherkin.md), Tags). When that run is green, `.github/workflows/claude-review.yml` runs the nm-review skill on the pull request and submits its verdict as an approval or a change request from github-actions[bot]. Only the first green run starts a review: a pull request that already has a review from github-actions[bot] waits for `/nm-review`. Set the Status of the issue to In review (section 7). Wait for CI (`gh pr checks --watch`), then for the review run (`gh run watch` on the latest `claude-review.yml` run), and read the verdict (`gh pr view <N> --json reviews`). Do not run the nm-review skill locally for this step: the approval from the workflow is the one the branch rules count.
4. **Fix and review again**: Fix every finding, put all fixes of the round in one commit, push, wait for CI (each push to a ready pull request runs the `@smoke` tests in Chromium), and comment `/nm-review` on the pull request (`gh pr comment <N> --body "/nm-review"`; `/nm-review full` also writes and runs the tests that prove its hypotheses) until the verdict is an approval. A push to main runs all the selected end-to-end tests and acceptance scenarios in Chromium and WebKit.
5. **Request CodeRabbit and merge**: CodeRabbit reviews only on request (`auto_review` is off in `.coderabbit.yaml`) and is limited to about five reviews an hour, so a pull request gets one request per round. Once CI is green and the review workflow has approved, the session that coordinates the pull requests triggers CodeRabbit (`gh pr comment <N> --body "@coderabbitai review"`) and wait for its review. Fix every finding or answer it, and resolve the threads. All fixes of one CodeRabbit round go in one commit, and that commit repeats step 4 (CI and `/nm-review`). When the findings are fixed or answered, the threads are resolved and step 4 is green again, comment `@coderabbitai review` once per round (after step 4 has approved that commit): CodeRabbit reviews the new commits and replaces its change request, which stays on the pull request until then. Request it again only after fixes, never after a plain push, and never before the github-actions approval. When every finding is answered without a pushed commit, a repeat request has no new commits to review and CodeRabbit may keep its change request: tell the owner and do not request again. A dismissed change request is not an approval: the owner decides how the second approval is obtained, and the agent does not merge. The branch rules need two approvals, github-actions and CodeRabbit: squash-merge by hand (`gh pr merge <N> --squash`) when both are in place. Auto-merge is never enabled (it merges before CodeRabbit answers) and `--admin` is never used (it skips the two approvals). An implementation agent stops at the github-actions approval, reports once and does not merge; the merge is done by whoever coordinates the pull requests.

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
| Milestone | the issue   | `YYYY.MM`, the release of that month        | the owner for the plan; the milestone workflow at a merge   |

A milestone is named after the month of its release and has the last day of the month as its due date. The milestone of an open issue is the month of its planned release. When the pull request merges in a later month, the workflow moves the issue to the month of the merge, because the earlier month was not released; a milestone of the same or a later month stays. A milestone is closed when its release is published.

An agent keeps the status of its issue true:

1. **Takes an issue**: sets Status to In Progress before the first commit. An agent works only on an issue whose Status is Todo or In Progress; an issue in Backlog is the owner's to move.
2. **Opens the pull request** (step 1 of the pipeline): when the issue has no milestone, sets the current milestone on it (a merged pull request gets it from the workflow). **Marks it ready** (step 3): sets Status to In review.
3. **Merge or close**: Done, set by the built-in project workflows (a closed issue and a merged pull request set it). An agent does not set Done by hand.
4. **Files a new issue**: sets its Type, adds it to the project with its Area, and leaves Status at Backlog; Priority and Effort stay with the owner. The labels of an issue only name what those fields do not: `perf`, `epic`, `needs-decision` and the area labels of [`CONTRIBUTING.md`](../../CONTRIBUTING.md), which are finer than the project field Area and are set beside it. Type is never a label.
5. **Changes the decision**: updates the issue at once (Problem, Cause, Change, Acceptance, and the title when it names the old design), notes in its Notes why the earlier plan was dropped, and updates the pull request body. The review workflow and CodeRabbit (`Linked Issues`, `Out of Scope`) compare the pull request with the issue text and report a deviation from an outdated issue as a defect.

The commands:

```
gh project item-add 15 --owner jiva-studio --url <issue url>
gh project item-edit --id <item id> --project-id PVT_kwDOBtvOi84Brz7D --field-id <Status field id> --single-select-option-id <option id>
```

The ids come from `gh project field-list 15 --owner jiva-studio --format json` and `gh project item-list 15 --owner jiva-studio --format json`.

An issue that needs several pull requests is split into sub-issues, one per pull request; each pull request closes its own sub-issue with `Closes #N`, and the parent closes when its last sub-issue does. A pull request is written with `Closes #N`, so the issue and the pull request move together. The workflow `milestone.yml` puts a merged pull request, the issues it closes and an issue closed as completed in the milestone of the month when they have none or have an earlier one.
