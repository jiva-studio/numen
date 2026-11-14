# Code Quality & Metrics Validator (`@numen/lint-quality`)

Checks over the whole repository that no single module's lint target covers: duplication and unused TypeScript and Vue code. Go complexity (`cyclop`, `gocognit`, `nestif`, `funlen`), interface size (`interfacebloat`) and unused parameters and results (`unparam`) are linters of `.golangci-structure.yml`, and ESLint and golangci-lint run in the lint target of each module.

The rule for duplication is in `tools/checks/structure/ts/duplication` and is listed in the registry of `tools/checks`; this command runs it together with knip. The comparison of the plugin API types is the Go check `go/api-types` of `tools/checks`.

## Invariants Checked

1. **Code Duplication (`jscpd`)**:
   - Scans handwritten code in one scope for `libs` and `apps` and one scope per plugin directory. Plugins are separate repositories that depend only on the SDK, so copies between plugins are not counted.
   - Ignores tests, fixtures, generated code, stories, and build artifacts.
   - Enforces a maximum duplication rate threshold per scope (default: <= 3.5%).

2. **Unused Files, Dependencies and Exports (`knip`)**:
   - Runs `knip` once with the root `knip.json`, whose workspaces name every TS package and plugin frontend. Unused files, unused and unlisted dependencies fail; unused exports and types are reported. A missing `knip` binary, a missing `knip.json` or a frontend package outside its workspaces fails the check.

---

## Usage

### Via Makefile

```bash
make check_quality                       # Run full parallel suite
make check_quality check=duplication     # Run only duplication check
make check_quality check=knip            # Run only knip
make check_quality opts="--warn"         # Non-blocking / advisory mode
```

### Directly with Node

```bash
# Run all checks in parallel
node tools/lint-quality/check.mjs

# Run specific checks
node tools/lint-quality/check.mjs --check=duplication,knip

# Run only on changed files (fast mode)
node tools/lint-quality/check.mjs --changed-only

# Advisory / warning mode (exit 0)
node tools/lint-quality/check.mjs --warn

# Output machine-readable JSON
node tools/lint-quality/check.mjs --json
```
