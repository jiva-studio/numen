# Code Quality & Metrics Validator (`@numen/lint-quality`)

Automated verification of code quality, duplication, complexity, and symbol hygiene across Go and TypeScript/Vue modules.

## Invariants Checked

1. **Code Duplication (`jscpd`)**:
   - Scans all handwritten code in `libs`, `apps`, and `plugins`.
   - Ignores tests, fixtures, generated code, stories, and build artifacts.
   - Enforces a maximum duplication rate threshold (default: <= 3.0%).

2. **Cyclomatic & Cognitive Complexity (`cyclop`, `gocognit`, `nestif`, `funlen`)**:
   - Enforces complexity boundaries on Go modules and plugins:
     - `cyclop`: cyclomatic complexity <= 15.
     - `gocognit`: cognitive complexity <= 15.
     - `nestif`: nesting depth <= 4.
     - `funlen`: <= 80 lines / 50 statements.

3. **Dead Code & Unused Symbols (`knip` & `unparam`)**:
   - **Frontend (TS/Vue)**: runs `knip` across UI modules to catch unused exported functions, constants, and types.
   - **Backend (Go)**: runs `unparam` to detect unused function parameters and dead return values.

4. **Interface Segregation & Sizing (`interfacebloat`)**:
   - Detects bloated Go interfaces that exceed 10 methods, encouraging focused, cohesive contracts.

---

## Usage

### Via Makefile

```bash
make check_quality                       # Run full parallel suite
make check_quality check=duplication     # Run only duplication check
make check_quality check=complexity      # Run only Go complexity check
make check_quality check=deadcode        # Run only dead code / unused symbols
make check_quality opts="--warn"         # Non-blocking / advisory mode
```

### Directly with Node

```bash
# Run all checks in parallel
node tools/lint-quality/check.mjs

# Run specific checks
node tools/lint-quality/check.mjs --check=duplication
node tools/lint-quality/check.mjs --check=complexity,deadcode

# Run only on changed files (fast pre-push mode)
node tools/lint-quality/check.mjs --changed-only

# Advisory / warning mode (exit 0)
node tools/lint-quality/check.mjs --warn

# Output machine-readable JSON
node tools/lint-quality/check.mjs --json
```
