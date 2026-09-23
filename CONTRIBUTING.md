# Contributing

## Layout

Every module is listed in the [README](README.md#layout), which is the one place they are written down.

The repo is mixed-language by design. Each module owns its own build and its own tests; the root carries no build system. The npm packages share one install, at `modules/`, because a package installed twice is two types that never match.

## Setup

```bash
cd modules && npm install
```

That is the whole setup. `modules/` is the npm workspace root, so one install fetches every package under it — the interface libraries, the two windows, the phone, the sites and the tools — and one copy of a package answers for all of them. It also installs commitlint, wires the `commit-msg` git hook through husky, and points `commit.template` at the message template.

The install lives in `modules/` rather than at the repository root, deliberately: the repo is mixed-language, and a `package.json` and `node_modules` in the root would read as "this is a Node project". The root carries no build system; `modules/` is the source root, and the Go modules under it are built by go.

## Issue labels

- **Type**: `task`, `bug`, `feature`, `adr`.
- **Area**: `desktop`, `mobile`, `core`, `domain`, `protocol`, `ui`, `wire`, `docs`, `ci`, `repo`.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org). The format is checked in three places: locally by the `commit-msg` hook, in CI over every commit in a pull request — so a clone that skipped `npm install` is still caught before merge — and on the **pull request title**.

The title matters because merges are squash-only: the title becomes the subject of the single commit that lands in `main`. A tidy branch behind a sloppy title still produces a sloppy history.

**A title has 64 characters.** The squash appends ` (#123)` to it, and the header the hook measures is what that comes to.

```
<type>(<scope>): <subject>

<body>

<footer>
```

- **subject** — imperative, no trailing dot, header capped at 72 characters.
- **types** — `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
- **scopes** — follow the module layout: `desktop`, `mobile`, `landing`, `core`, `domain`, `protocol`, `ui`, `wire`, `adr`, `docs`, `vault`, `ci`, `deps`, `repo`. An unlisted scope is a **warning**, not an error: adding a module should never be blocked by a forgotten line in `modules/tools/git-hooks/commitlint.config.mjs` — but a typo still shows up.
- **breaking change** — `!` after the scope, or a `BREAKING CHANGE:` footer.

```
docs(adr): add a record on two levels of addressing
feat(domain): resolve wikilinks by name with priority rules
fix(protocol): keep note:// links untouched on rename
feat(domain)!: drop path-based note identity
```

Run the check by hand against the last commit:

```bash
cd modules/tools/git-hooks && npx commitlint --last --verbose
```

## Checks on other platforms

The core suite runs on Linux for every pull request. A Windows runner costs twice a Linux one and a macOS runner ten times, so those two run only when a commit in the branch carries a `Run-On:` trailer, or when the **Core** workflow is started by hand with the platform ticked. Every commit in the branch is read for the trailer, so it goes on the commit that needed the platform.

```
fix(core): an append that lands short is taken back

Run-On: macos
```

The values are `windows`, `macos`, the two of them separated by a comma, or `all`. What Windows and macOS say is reported and does not hold the pull request.

## Architecture decisions

Decisions and technical constraints live in [`docs/adr/`](docs/adr/README.md), which documents the rationale, architectural boundaries, and design trade-offs. Before proposing significant architectural changes, review existing ADRs or propose a new one.

## Development workflow

1. **Fork and branch**: Create a feature branch from `main`.
2. **Commit conventions**: Use [Conventional Commits](https://www.conventionalcommits.org) format (`<type>(<scope>): <subject>`).
3. **Run checks locally**:
   - `make lint` — run linters across Go and TypeScript.
   - `make test` — run unit and integration test suites.
   - `make desktop` — build the desktop application locally.
4. **Submit Pull Request**: Open a pull request against `main` with a clear description of the problem solved and testing steps performed.
