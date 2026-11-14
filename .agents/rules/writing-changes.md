# Pull requests, commits and issues

How a pull request, its commit and an issue are written. The reader is a person who has not read the code.

## Pull request and commit

Merges are squash only. The pull request title becomes the commit subject and the pull request body becomes the commit body, so the body is written once, in the pull request.

**Title.** Conventional Commits: `<type>(<scope>): <subject>`, imperative, no trailing dot. CI checks the title (`tools/git-hooks/check-commit.mjs`); the types and scopes are in [`CONTRIBUTING.md`](../../CONTRIBUTING.md).

**Body.** Three sections, then the issue it closes:

- `## Problem` — what a user or developer saw, in plain words.
- `## Cause` — why it happened, in one or two sentences. A `feat` pull request may leave this section out: a new capability has a need, not a cause.
- `## Change` — what happens now.
- Last line: `Closes #N`.

Write each paragraph on one line. Name an identifier only when the reader needs it. Test output, counts of passing tests and the list of commands that were run go in the review conversation, not in the body.

Example:

```markdown
## Problem

The core call that returns the notes related to one note is named `GetNeighborhood`. A reader of the proto or of a plugin cannot tell from the name what the data is.

## Cause

The name comes from the picture the plex draws, not from the data.

## Change

The call is `ListRelatedNotes` and its items are `RelatedNote`, in the proto, the kernel, both SDKs and the plugins that use it.

Closes #1838
```

Commits on a branch are squashed away, so only their subject line follows the title format.

## Issue

Title: `<Area>: <what>`, for example `Search: show the date of a result`. Body:

- `## Problem` — what is missing or wrong.
- `## Cause` — for a bug, when it is known. A feature may leave it out.
- `## Change` — what should happen instead.
- `## Acceptance` — a list of observable results that mean the issue is done.
- `## Notes` — optional: related issues, order, open questions.

A bug also says how to reproduce it and what was expected, under Problem. One issue is one piece of work; work that needs "and" in its title is two issues.

The issue forms in [`.github/ISSUE_TEMPLATE`](../../.github/ISSUE_TEMPLATE) and the pull request template in [`.github/PULL_REQUEST_TEMPLATE.md`](../../.github/PULL_REQUEST_TEMPLATE.md) use these sections. `gh issue create` and `gh pr create` skip the forms, so a body written from the command line follows this file.
