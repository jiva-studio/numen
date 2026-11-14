# Comment placement check (`style/comments` of `tools/checks`)

Rejects comments that repeat what a name and a type already state. The rule is §1 of `AGENTS.md`.

| Language        | Rejected                                                                                                                                                                                                               |
| --------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go              | a comment on a struct field (above it or at the end of its line); a doc comment on an unexported function or method; a doc comment, of any length, whose words are the name and the signature of its declaration (function, method, type, constant or variable), also with a blank line between them |
| TypeScript, Vue | a comment on an interface, type literal or class property, or on a member of the object passed to `defineProps` or `defineEmits`; a doc comment on a top-level function that is not exported and on a private method; a doc comment, of any length, whose words are the name and the signature of its declaration (function, method, type, variable), also with a blank line between them |

A doc comment restates its declaration when 80 percent or more of its words, stop words (`a`, `the`, `of`, `is`, ...) left out, are words of the names: the identifier, the package (Go) or file name, the receiver, and the names and types of the parameters and results, each split at camelCase humps and read without plural endings. Verbs that mean one operation count as one word (`get`, `find`, `read`, `fetch`, `return`, `load` and `set`, `store`, `save`, `write`, `update`; `create`, `make`, `build`; `delete`, `remove`), and a comment word that starts a name word (or the reverse) of four letters or more matches it (`window` and `windowsession`). The threshold, the stop words and the synonyms are defined once, in `restate.json`, which `restate.go` embeds and `check.mjs` reads. `ReadTheme reads the stylesheet of a theme`, `SetVault sets the vault the window shows` and `GetPlugin finds no plugin` are refused; a comment that names a fact the signature does not (`... from the bundled copy when the vault holds none`) stays. A comment that is detached by a blank line counts when it starts with the identifier (Go) or is a JSDoc block (TypeScript). `--fix` removes the reported comments.

Not reported:

- exported fields of a Go type whose doc comment carries a marker line: `//numen:api` for a type of the plugin SDK, whose field comments plugin authors read in the SDK documentation, and `//numen:settings` for a settings type, whose field comments `apps/docs/tools/manual.mjs` writes into the settings reference (the manual refuses a settings type without the marker);
- directives: `//go:`, `//export`, `gomutants:`, `Deprecated:`, `eslint`, `@ts-`, `Stryker`, `istanbul`, `prettier`, `@deprecated`;
- section banners (`/* ---- Handlers ---- */`) and, for the field and function rules, a comment separated from the declaration by a blank line;
- generated files (`Code generated ... DO NOT EDIT` or `@generated` in the first lines);
- `.proto` files and plain JavaScript.

Go files are parsed with `go/ast` by `main.go` (standard library only); TypeScript and Vue files with `typescript` and `@vue/compiler-sfc` by `check.mjs`.

## Usage

```bash
node tools/checks/style/comments/check.mjs                 # every tracked file (CI: make lint_comments)
node tools/checks/style/comments/check.mjs --staged        # staged files
node tools/checks/style/comments/check.mjs a.go b.vue      # the files named
node tools/checks/style/comments/check.mjs --fix a.go      # remove the reported comments
make checks_selftest ONLY=style/comments          # fixtures of both checkers
```
