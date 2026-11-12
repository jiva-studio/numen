# ESLint and stylelint rules (`@numen/lint-layers`)

The rule files the packages lint with: `base-eslint.mjs` (`createBaseConfig` and the strict rule sets), `numen-rules.mjs` (the `numen/*` ESLint rules, copied byte for byte into every plugin), `layers.mjs` (the layer policies of the windows and libraries), `stylelint-plugin.mjs` (the stylelint rules of `stylelint.config.mjs`) and `dependency-cruiser.mjs` (the dependency-cruiser configuration).

The tests that prove the rules are in this folder (`npm test`) and are registered in `tools/checks` with the runners `eslint` and `stylelint`. The checks that apply them to the repository, and the structural checks of the Go and TypeScript code, are in `tools/checks` (`make checks`).
