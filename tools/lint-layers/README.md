# Architectural Layer & Boundary Validator (`@numen/lint-layers`)

Automated verification of architectural layer boundaries and invariants across
libraries, plugins, and desktop applications.

## Invariants Checked

1. **Libraries (`libs`)**:
   - Go Hexagonal Architecture: core purity, inward dependencies, no machine ports in domain.
   - Depguard rules: core, notes, and plugin SDK Go isolation.
   - UI library isolation: zero domain, protocol, wire, or application imports.
   - The core holds no package a domain plugin owns (`usecase/document`).
2. **Plugins (`plugins`)**:
   - Sandbox & isolation: zero cross-plugin imports, zero core imports.
   - Manifest validation: valid ID, name, version.
   - Frontend boundaries: zero `@/**` or `@plugins/**` imports; contract only.
   - No plugin reaches `apps/`: no relative path, alias or import into it.
3. **Applications (`apps`)**:
   - Feature-Sliced Design (FSD) layer hierarchy (`shared` → `entities` → `features` → `widgets` → `pages` → `app`).
   - No application reaches `plugins/`: no relative path, alias, `import.meta.glob`, tsconfig path or Go import into it. A path is resolved from the file and from every package root above it.

## Usage

```bash
# Via Makefile:
make check_layers                     # all layers across monorepo
make check_layers target=libs         # library layers only
make check_layers target=plugins      # plugins isolation and boundaries
make check_layers target=apps         # application FSD layers
make check_layers target=core         # specific module

# Directly:
node check.mjs [target...]
node check.mjs --list
node check.mjs --help
```
