# `@numen/plugin-sdk`

The conventions shared across the repository are in `AGENTS.md` at the root. This file defines the rules for `libs/plugin/sdk-ts`. What the SDK holds, who may depend on it and that it is published are rules (c), (d), (f) and (j) of [`.agents/rules/architecture.md`](../../../.agents/rules/architecture.md); the placement skill is [`nm-place-plugin`](../../../.agents/skills/nm-place-plugin/SKILL.md).

## Exports

An entry is a key of `exports` in `package.json`, and a plugin names only these:

- `.` is the main entry: `src/index.ts` is a list of named exports, never `export *`, so each name of the public contract is written once and reviewed when it is added.
- `./contracts` holds the contracts between plugins, one namespace per contract. The main entry does not carry them.
- `./testing` and `./contracts/testing` hold test code: the mock hosts and the servers of the contracts. No other entry reaches a `testing` folder, so no test code ships in the bundle the window serves at `/sdk/plugin-sdk.js` (check `no-testing-module-in-entry`).
- `./host-modules` and `./vite` are for the loader and the build, not for plugin code.

A new export of the main entry has a consumer in a plugin or the loader. A helper that a plugin calls stays public.

## No wire type in the public surface

No exported declaration mentions a declaration of `@numen/protocol` (check `sdk-ts-encapsulates-protocol`). The SDK maps a message to its own type, and a code of the schema to a name of its own. `@numen/protocol` is a dependency, because the SDK imports it at run time.

## Nothing happens at import

A module creates no transport and no client when it loads. A client is made in the function that calls it.

## Names and types

A function is an imperative verb phrase (`getAssetUrl`, `createApi`, `formatFingerprint`) or a predicate (`isWebAddress`). A caught value or a value read from outside is `unknown`, narrowed where it is read; the SDK has no alias of it.
