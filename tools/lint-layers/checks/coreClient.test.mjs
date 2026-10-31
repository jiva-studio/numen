import assert from "node:assert/strict";
import { test } from "node:test";
import { findDisallowedHostImports } from "./frontend.mjs";
import {
  PROTOCOL_USERS,
  findCoreClientPublishViolations,
  findProtocolViolations,
  readProtocolReferences,
} from "./coreClient.mjs";
import { findPluginForbiddenPackageUses } from "./plugins.mjs";

const reference = (owner, symbol = 'import "@numen/protocol"') => ({ file: `libs/${owner}/src/a.ts`, line: 3, owner, symbol });

test("only the plugin SDK and the core client may name the schema", () => {
  assert.deepEqual(PROTOCOL_USERS, ["@numen/plugin-sdk", "@numen/core-client"]);
});

test("the SDK and the core client importing the schema are clean", () => {
  assert.deepEqual(findProtocolViolations([reference("@numen/plugin-sdk"), reference("@numen/core-client")]), []);
});

test("an application, the plugin host and a plugin that name the schema are reported with the line", () => {
  const found = findProtocolViolations([reference("@numen/editor"), reference("@numen/plugin-host"), reference("@numen/readers-pdf")]);
  assert.equal(found.length, 3);
  assert.match(found[0].symbol, /@numen\/editor names @numen\/protocol/);
  assert.equal(found[0].line, 3);
});

test("a declaration of the schema in a package.json is a reference like an import", () => {
  const found = findProtocolViolations([
    { file: "apps/mobile/package.json", line: 1, owner: "@numen/mobile", symbol: "dependencies: @numen/protocol" },
  ]);
  assert.equal(found.length, 1);
  assert.equal(found[0].file, "apps/mobile/package.json");
});

test("the schema references of this repository are read, and only the SDK and the core client hold any", () => {
  const { references, examined, packages } = readProtocolReferences();
  assert.ok(packages >= 15);
  assert.ok(examined >= 500);
  assert.ok(references.filter((one) => PROTOCOL_USERS.includes(one.owner)).length >= 10);
  assert.deepEqual(findProtocolViolations(references), []);
});

const PRIVATE_MANIFEST = { name: "@numen/core-client", private: true };
const PUBLISH = { file: ".github/workflows/release.yml", text: "steps:\n  - run: npm publish --access public\n" };

test("a private core client that no publishing workflow names is clean", () => {
  assert.deepEqual(findCoreClientPublishViolations(PRIVATE_MANIFEST, [PUBLISH]), []);
});

test("a core client that is not private is reported", () => {
  const found = findCoreClientPublishViolations({ name: "@numen/core-client" }, []);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /not "private": true/);
});

test("a publishing workflow that names the core client is reported with the line", () => {
  const workflow = { file: ".github/workflows/release.yml", text: "steps:\n  - run: npm publish --workspace libs/core-client\n" };
  const found = findCoreClientPublishViolations(PRIVATE_MANIFEST, [workflow]);
  assert.equal(found.length, 1);
  assert.equal(found[0].line, 2);
});

test("a workflow that names the core client without publishing is clean", () => {
  const workflow = { file: ".github/workflows/desktop-frontend.yml", text: "paths:\n  - libs/core-client/**\n  - run: npm test\n" };
  assert.deepEqual(findCoreClientPublishViolations(PRIVATE_MANIFEST, [workflow]), []);
});

test("a plugin package.json that depends on the core client or the schema is reported", () => {
  const text = JSON.stringify({ dependencies: { "@numen/plugin-sdk": "*", "@numen/core-client": "*" }, devDependencies: { "@numen/protocol": "*" } });
  const found = findPluginForbiddenPackageUses("plugins/a/package.json", text);
  assert.deepEqual(found.map((one) => one.symbol), ["dependencies: @numen/core-client", "devDependencies: @numen/protocol"]);
});

test("a plugin source that imports the core client, a path of it or the schema is reported", () => {
  const text = [
    "import { vault } from '@numen/core-client'",
    "import { transport } from '@numen/core-client/testing'",
    "import type { Entry } from '@numen/protocol'",
    "import { definePlugin } from '@numen/plugin-sdk'",
  ].join("\n");
  const found = findPluginForbiddenPackageUses("plugins/a/frontend/index.ts", text);
  assert.deepEqual(found.map((one) => one.line), [1, 2, 3]);
});

test("a plugin source that imports only the SDK and the UI is clean", () => {
  const text = "import { definePlugin } from '@numen/plugin-sdk'\nimport { Button } from '@numen/ui'\n";
  assert.deepEqual(findPluginForbiddenPackageUses("plugins/a/frontend/index.ts", text), []);
});

test("the plugin host may import vue, the SDK and the core client, and its own files", () => {
  const specifiers = ["vue", "@numen/plugin-sdk", "@numen/plugin-sdk/host-modules", "@numen/core-client", "@numen/core-client/testing", "./loader"];
  assert.deepEqual(findDisallowedHostImports("loader.ts", specifiers), []);
});

test("the plugin host importing the schema, the UI or the shell is reported", () => {
  const found = findDisallowedHostImports("loader.ts", ["@numen/protocol", "@numen/ui", "@numen/shell", "@bufbuild/protobuf"]);
  assert.equal(found.length, 4);
  assert.match(found[0], /libs\/plugin-host-ts\/src\/loader\.ts: @numen\/protocol/);
});

test("the plugin host tests may import vitest, and its build entry vite", () => {
  assert.deepEqual(findDisallowedHostImports("loader.test.ts", ["vitest"]), []);
  assert.deepEqual(findDisallowedHostImports("vite.ts", ["vite"]), []);
  assert.equal(findDisallowedHostImports("loader.ts", ["vitest"]).length, 1);
});
