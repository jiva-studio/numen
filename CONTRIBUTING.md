# Contributing

## Layout

Every module is listed in the [README](README.md#layout), which is the one place they are written down.

The repo is mixed-language by design. Each module owns its own build and its own tests; the root carries no build system. The npm packages share one install, at `modules/`, because a package installed twice is two types that never match.

## Setup

```bash
cd modules && npm install
```

That is the whole setup. `modules/` is the npm workspace root, so one install fetches every package under it — the interface libraries, the two windows, the phone, the sites and the tools — and one copy of a package answers for all of them. It also wires git hooks through husky and points `commit.template` at the message template.

The install lives in `modules/` rather than at the repository root, deliberately: the repo is mixed-language, and a `package.json` and `node_modules` in the root would read as "this is a Node project". The root carries no build system; `modules/` is the source root, and the Go modules under it are built by go.

## Issue labels

An issue carries exactly one type and at least one area.

- **Type**: `bug` (something is broken: a wrong result, a crash, a leak), `perf` (it works, but is too slow or too heavy), `feature` (new behaviour), `task` (implementation work that is neither), `epic` (tracks a set of issues).
- **Area**: `desktop`, `mobile`, `core`, `domain`, `protocol`, `ui`, `wire`, `vault`, `docs`, `ci`, `repo`.
- **Status**: `needs-decision` (blocked on a decision, not on work).

Dependabot sets `dependencies`, `go`, `javascript` and `github_actions` on its own pull requests.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org).

Merges are squash-only: the PR title becomes the subject of the single commit that lands in `main`.

```
<type>(<scope>): <subject>

<body>

<footer>
```

- **subject** — imperative, no trailing dot.
- **types** — `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- **scopes** — follow the module layout: `desktop`, `mobile`, `landing`, `core`, `domain`, `protocol`, `ui`, `wire`, `docs`, `vault`, `ci`, `deps`, `repo`.
- **breaking change** — `!` after the scope, or a `BREAKING CHANGE:` footer.

```
feat(domain): resolve wikilinks by name with priority rules
fix(protocol): keep note:// links untouched on rename
feat(domain)!: drop path-based note identity
```

## Checks on other platforms

The core suite runs on Linux for every pull request. A Windows runner costs twice a Linux one and a macOS runner ten times, so those two run only when a commit in the branch carries a `Run-On:` trailer, or when the **Core** workflow is started by hand with the platform ticked. Every commit in the branch is read for the trailer, so it goes on the commit that needed the platform.

```
fix(core): an append that lands short is taken back

Run-On: macos
```

The values are `windows`, `macos`, the two of them separated by a comma, or `all`. What Windows and macOS say is reported and does not hold the pull request.

## Architecture

Architecture conventions and technical constraints live in [`AGENTS.md`](AGENTS.md) and [`.agents/rules/`](.agents/rules/). Before proposing a significant architectural change, read those and update them in the same pull request that changes what they describe.

## Development workflow

1. **Fork and branch**: Create a feature branch from `main`.
2. **Commit conventions**: Use [Conventional Commits](https://www.conventionalcommits.org) format (`<type>(<scope>): <subject>`).
3. **Run checks locally**:
   - `make lint` — run linters across Go and TypeScript.
   - `make test` — run unit and integration test suites.
   - `make desktop` — build the desktop application locally.
4. **Submit Pull Request**: Open a pull request against `main` with a clear description of the problem solved and testing steps performed.
