import assert from "node:assert/strict";
import { test } from "node:test";
import { PROTOCOL_USERS, findCoreClientPublishViolations, findPluginReliedPackages, findProtocolDeclarations } from "./check.mjs";

const manifest = (file, name, fields = {}) => ({ file, manifest: { name, ...fields } });

test("only the plugin SDK and the core client may declare the schema", () => {
  assert.deepEqual(PROTOCOL_USERS, ["@numen/plugin-sdk", "@numen/core-client"]);
});

test("a declaration of the schema by a package other than the two clients is reported with the field", () => {
  const found = findProtocolDeclarations([
    manifest("libs/plugin/sdk-ts/package.json", "@numen/plugin-sdk", { dependencies: { "@numen/protocol": "*" } }),
    manifest("apps/mobile/package.json", "@numen/mobile", { devDependencies: { "@numen/protocol": "*" } }),
  ]);
  assert.equal(found.length, 1);
  assert.equal(found[0].file, "apps/mobile/package.json");
  assert.match(found[0].symbol, /@numen\/mobile declares @numen\/protocol in devDependencies/);
});

const CLIENT = manifest("libs/core/client/package.json", "@numen/core-client", { private: true, dependencies: { "@numen/protocol": "*" } });
const SDK = manifest("libs/plugin/sdk-ts/package.json", "@numen/plugin-sdk", { dependencies: { "@numen/protocol": "*" }, devDependencies: { "@numen/core-client": "*" } });
const PROTOCOL = manifest("libs/core/protocol/package.json", "@numen/protocol");
const PLUGIN = manifest("plugins/a/package.json", "@numen/a", { dependencies: { "@numen/plugin-sdk": "*" } });

test("the packages plugins rely on are what they declare and what that brings when installed", () => {
  assert.deepEqual([...findPluginReliedPackages([CLIENT, SDK, PROTOCOL, PLUGIN])].sort(), ["@numen/plugin-sdk", "@numen/protocol"]);
});

test("a private core client that plugins do not rely on is clean", () => {
  assert.deepEqual(findCoreClientPublishViolations([CLIENT, SDK, PROTOCOL, PLUGIN]), []);
});

test("a core client that is not private is reported", () => {
  const open = manifest(CLIENT.file, "@numen/core-client");
  assert.match(findCoreClientPublishViolations([open, SDK, PROTOCOL, PLUGIN])[0].symbol, /not "private": true/);
});

test("a core client that a package plugins rely on brings in is reported", () => {
  const bringing = manifest(SDK.file, "@numen/plugin-sdk", { peerDependencies: { "@numen/core-client": "*" } });
  assert.match(findCoreClientPublishViolations([CLIENT, bringing, PROTOCOL, PLUGIN])[0].symbol, /plugins rely on @numen\/core-client/);
});

test("plugins that rely on no first-party package fail the rule", () => {
  assert.match(findCoreClientPublishViolations([CLIENT, SDK, PROTOCOL])[0].symbol, /the published set is empty/);
});

test("a missing core client package is reported", () => {
  assert.equal(findCoreClientPublishViolations([SDK, PLUGIN]).length, 1);
});
