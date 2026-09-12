# TypeScript and Vue are formatted by hand

- **Status:** Accepted
- **Date:** 2026-09-12
- **Applies to:** `modules/apps/desktop`, `modules/apps/mobile`, `modules/libs/ui`, `modules/tools`
- **Related:** [How this application is tested](0025-how-this-application-is-tested.md), [A file of the windows stands on a layer](0042-a-file-of-the-windows-stands-on-a-layer.md)

## Context

No formatter runs over the TypeScript and Vue here. There is no Prettier, no `.prettierrc`, no `format` script, and no formatting step in the hook or the workflow. What holds the files to one shape is `.editorconfig` for the bytes every editor agrees on, ESLint for what is an error, and the hand of whoever wrote the line.

Go is the other half of this repository and is not in the same position: `gofmt` is the language's own formatter, it ships with the toolchain, every Go reader expects its output, and `gofmt -l` is run as a check. The question this record settles is why the other half is not treated the same way.

The reason is what these files are. A line here is written to be read: a signature broken where the argument list stops being one thought, a table of words laid out as a table, a `computed` kept on one line because it is one idea. A formatter is a function of the token stream and cannot see any of that, so it rewrites those lines into its own shape — and the shape it picks is right often enough that the places it is wrong are the places a reader was being helped.

The second reason is the diff. Every change here is read by a person before it lands, and a formatter run turns a two-line change into a two-hundred-line one. That cost is paid once on adoption and then again, quietly, every time a tool's minor version moves its own defaults.

## Decision

### Prettier is not installed, and no formatter is run over TypeScript, Vue or Markdown

No package here depends on Prettier, and none declares a `format` script. A pull request that adds one is a change to this record.

### What holds a file to a shape

- `.editorconfig` — encoding, line endings, the final newline, trailing whitespace, and the indent. Every editor reads it, and it settles the things nobody has an opinion about.
- ESLint, with the house rules in `modules/tools/lint` beside it — what is wrong, not what is pretty. A rule there refuses a thing a reader would trip over; it never restates a preference about where a line breaks.
- The reviewer. Layout is read like the rest of the code.

### Go is formatted by `gofmt`, and that is not an exception to this

`gofmt` is part of the language rather than a choice made about it: there is one output, it has never moved, and Go source that has not been through it reads as wrong to every Go reader. Nothing above applies to it. `gofmt -l` printing a name is a failure.

### Generated files are whatever generated them

`protoc-gen-go`, `protoc-gen-connect-go` and `protoc-gen-es` write their own shape, and nothing here touches it. A generated file is not hand-formatted, not linted for layout, and not read for style.

## Consequences

- **Layout is a reviewer's business.** A line laid out badly is raised the way a name chosen badly is raised.
- **A diff is what changed.** Nothing lands carrying lines nobody edited, and `git blame` reaches the person who wrote a line rather than the run that reflowed it.
- **A new file is written in the shape of the files beside it.** There is no command that will do it afterwards.
- **An editor with format-on-save configured for this repository will fight it**, and the fault will read as a large unrelated diff. Turn it off for these trees.
- **An agent working here formats by hand too**, and a wholesale reformat is a change nobody asked for.

## Alternatives considered

**Prettier with a shared config.** Rejected. It is the thing this record is about: it cannot see why a line was broken where it was, and adopting it rewrites every file once and then again on each of its own default changes.

**Prettier for new files only.** Rejected. Two shapes in one tree is worse than either, and "new" is not a property a check can read off a file.

**ESLint's own layout rules, or `@stylistic`.** Rejected for the same reason, at lower value: it is a formatter with a longer config, and every rule added is a preference argued over in a config file rather than in review.

**`dprint` or Biome.** Rejected. The objection is to a formatter, not to Prettier's implementation of one.
