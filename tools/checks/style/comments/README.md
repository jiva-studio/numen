# Comment placement check (`style/comments` of `tools/checks`)

Rejects comments that repeat what a name and a type already state. The rule is §1 of `AGENTS.md`.

| Language        | Rejected                                                                                                                                                                                                               |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go              | a comment on a struct field (above it or at the end of its line); a doc comment on an unexported function or method; a doc comment of more than five lines on an exported function, method, type, constant or variable |
| TypeScript, Vue | a comment on an interface, type literal or class property, or on a member of the object passed to `defineProps` or `defineEmits`; a doc comment on a top-level function that is not exported and on a private method   |

The limit is one constant per checker (`maxDocLines` in `main.go`, `MAX_DOC_LINES` in `check.mjs`). It counts the lines of text of the doc comment, blank comment lines included and directive and marker lines left out; the field comments of a `//numen:api` or `//numen:settings` type are not part of its doc. The message is "A doc comment is at most five lines: say the fact the name and type cannot show". `--fix` does not touch it: the facts to keep are the author's call.

Not reported:

- exported fields of a Go type whose doc comment carries a marker line: `//numen:api` for a type of the plugin SDK, whose field comments plugin authors read in the SDK documentation, and `//numen:settings` for a settings type, whose field comments `apps/docs/tools/manual.mjs` writes into the settings reference (the manual refuses a settings type without the marker);
- directives: `//go:`, `//export`, `gomutants:`, `Deprecated:`, `eslint`, `@ts-`, `Stryker`, `istanbul`, `prettier`, `@deprecated`;
- section banners (`/* ---- Handlers ---- */`) and a comment separated from the declaration by a blank line;
- generated files (`Code generated ... DO NOT EDIT` or `@generated` in the first lines);
- `.proto` files and plain JavaScript.

Go files are parsed with `go/ast` by `main.go` (standard library only); TypeScript and Vue files with `typescript` and `@vue/compiler-sfc` by `check.mjs`.

## Usage

```bash
node tools/checks/style/comments/check.mjs                 # every tracked file (CI: make lint_comments)
node tools/checks/style/comments/check.mjs --staged        # staged files (pre-commit hook)
node tools/checks/style/comments/check.mjs a.go b.vue      # the files named
node tools/checks/style/comments/check.mjs --fix a.go      # remove the reported comments
make checks_selftest ONLY=style/comments          # fixtures of both checkers
```
