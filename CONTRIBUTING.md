# Contributing

## Layout

```
docs/adr/                    architecture decision records
modules/apps/desktop/        desktop client
modules/apps/mobile/         mobile client
modules/libs/domain/         domain model
modules/libs/protocol/       wire/vault protocol
modules/tools/git-hooks/     repo-level tooling (commit validation)
```

The repo is mixed-language by design. Each module owns its toolchain; the root
carries no build system.

## Setup

```bash
cd modules/tools/git-hooks && npm install
```

That is the whole setup for repo-level tooling. It installs commitlint, wires the
`commit-msg` git hook through husky, and points `commit.template` at the message
template. Modules under `modules/` carry their own toolchains.

The Node install lives in `modules/tools/git-hooks/` rather than at the repo
root, deliberately: the repo is mixed-language, and a `package.json` +
`node_modules` in the root would read as "this is a Node project". Nothing
outside that directory depends on Node.

## Labels

Two dimensions, and nothing else. A label answers *what this is* or *what part of
the repo it touches* — never *when it should be done*.

- **Type**, exactly one: `adr`, `task`, `bug`, `epic`.
- **Area**, zero or more: `desktop`, `mobile`, `domain`, `protocol`, `vault`,
  `repo`. These are the same words as the commit scopes, on purpose — one
  vocabulary for commits, issues and pull requests.
- **`needs-decision`** — the only exception: the issue is blocked on a decision,
  not on work.

Priority and status live in the project board as fields, not as labels. A label
cannot be sorted, cannot be a column, and turns into archaeology the moment
someone forgets to remove it.

## Commit messages

[Conventional Commits](https://www.conventionalcommits.org). The format is
checked in three places: locally by the `commit-msg` hook, in CI over every
commit in a pull request — so a clone that skipped `npm install` is still caught
before merge — and on the **pull request title**.

The title matters because merges are squash-only: the title becomes the subject
of the single commit that lands in `main`. A tidy branch behind a sloppy title
still produces a sloppy history.

```
<type>(<scope>): <subject>

<body>

<footer>
```

- **subject** — imperative, no trailing dot, header capped at 72 characters.
- **types** — `feat`, `fix`, `docs`, `refactor`, `perf`, `test`, `build`, `ci`,
  `chore`, `revert`.
- **scopes** — follow the module layout: `desktop`, `mobile`, `domain`,
  `protocol`, `adr`, `docs`, `vault`, `ci`, `deps`, `repo`. An unlisted scope is
  a **warning**, not an error: adding a module should never be blocked by a
  forgotten line in `modules/tools/git-hooks/commitlint.config.mjs` — but a typo
  still shows up.
- **breaking change** — `!` after the scope, or a `BREAKING CHANGE:` footer.

```
docs(adr): add ADR-0009 on two levels of addressing
feat(domain): resolve wikilinks by name with priority rules
fix(protocol): keep note:// links untouched on rename
feat(domain)!: drop path-based note identity
```

Run the check by hand against the last commit:

```bash
cd modules/tools/git-hooks && npx commitlint --last --verbose
```

## Architecture decisions

Decisions live in [`docs/adr/`](docs/adr/README.md) — the index has the reading
order and the rules for adding one. Progress is tracked in
[issue #13](https://github.com/jiva-studio/numen/issues/13).
