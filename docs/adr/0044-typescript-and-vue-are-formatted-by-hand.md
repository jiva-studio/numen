# Code formatting with Prettier and gofmt

- **Status:** Accepted
- **Date:** 2026-09-12
- **Applies to:** `modules/apps/desktop`, `modules/apps/mobile`, `modules/libs/ui`, `modules/tools`, `modules/libs/core`
- **Related:** [How this application is tested](0025-how-this-application-is-tested.md), [A file of the windows stands on a layer](0042-a-file-of-the-windows-stands-on-a-layer.md)

## Context

Code layout and formatting should be deterministic, fast, and automated across all modules so developers and agents focus entirely on logic, architecture, and correctness rather than cosmetic trivia.

The repository spans two main language ecosystems: Go for backend services and core logic, and TypeScript/Vue/CSS for frontend interfaces.

## Decision

### TypeScript, Vue, CSS, JSON, and Markdown are formatted with Prettier

Prettier is configured at the root repository level with `prettier-plugin-tailwindcss` to guarantee consistent code layout, HTML/Vue template formatting, and automatic Tailwind CSS utility class ordering.

All frontend packages declare a standard format check:
```bash
npm run format       # format all files
npm run format:check # verify formatting in CI
```

### Protocol-generated files are strictly ignored by Prettier

Code generated from Protobuf schemas (`modules/libs/protocol/src/numen/v1/**` and Go protobuf bindings) is owned strictly by `buf generate` / `protoc-gen-es` / `protoc-gen-go`.

These paths are declared in `.prettierignore` to prevent formatters from modifying machine-generated output or interfering with `buf generate` consistency checks.

### Go is formatted by `gofmt`

Go source is formatted with the standard Go toolchain via `gofmt`. Every Go file must match `gofmt -s` output, verified by `gofmt -l` in CI workflows.

## Consequences

- Automated formatters run across all files before committing and in CI.
- Diffs remain focused strictly on semantic changes and feature logic.
- Generated protocol files remain untouched and strictly reproducible from schemas.
