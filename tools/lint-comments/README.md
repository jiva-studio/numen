# Comment placement check (`@numen/lint-comments`)

Rejects comments that repeat what a name and a type already state. The rule is §1 of `AGENTS.md`.

| Language        | Rejected                                                                                                                                                                                                             |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Go              | a comment on a struct field (above it or at the end of its line); a doc comment on an unexported function or method                                                                                                  |
| TypeScript, Vue | a comment on an interface, type literal or class property, or on a member of the object passed to `defineProps` or `defineEmits`; a doc comment on a top-level function that is not exported and on a private method |

Not reported:

- exported fields of `libs/plugin-sdk-go` and properties of `libs/plugin-sdk-ts` outside tests;
- directives: `//go:`, `//export`, `gomutants:`, `Deprecated:`, `eslint`, `@ts-`, `Stryker`, `istanbul`, `prettier`, `@deprecated`;
- section banners (`/* ---- Handlers ---- */`) and a comment separated from the declaration by a blank line;
- generated files (`Code generated ... DO NOT EDIT` or `@generated` in the first lines);
- `.proto` files and plain JavaScript.

Go files are parsed with `go/ast` by `main.go` (standard library only); TypeScript and Vue files with `typescript` and `@vue/compiler-sfc` by `check.mjs`.

## Usage

```bash
node tools/lint-comments/check.mjs                 # every tracked file (CI: make lint_comments)
node tools/lint-comments/check.mjs --staged        # staged files (pre-commit hook)
node tools/lint-comments/check.mjs a.go b.vue      # the files named
node tools/lint-comments/check.mjs --fix a.go      # remove the reported comments
npm test --prefix tools/lint-comments              # fixtures of both checkers
```
