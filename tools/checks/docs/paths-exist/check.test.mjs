import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findMissingPaths } from "./check.mjs";

const TRACKED = [
  "libs/core/kernel/go.mod",
  "libs/core/kernel/container/layers_test.go",
  "libs/plugin/loader/src/index.ts",
  "plugins/views-tasks/manifest.json",
  "plugins/views-files/manifest.json",
  "tools/lint-layers/README.md",
  ".agents/rules/architecture.md",
];

const missing = (text) => findMissingPaths([{ file: "AGENTS.md", text }], TRACKED).findings.map((one) => `${one.line} ${one.symbol}`);

describe("the paths documents name exist in the tree", () => {
  it("reports a file and a folder that are not in the tree, with the line", () => {
    assert.deepEqual(missing("plain\nSee `libs/core/kernel/port` and `libs/core/kernel/go.mod`.\nAlso `tools/lint-layers/gone.mjs:12`."), [
      "2 libs/core/kernel/port is in no file or folder of the tree",
      "3 tools/lint-layers/gone.mjs:12 is in no file or folder of the tree",
    ]);
  });

  it("passes files, folders, a trailing slash, a line suffix and an anchor", () => {
    assert.deepEqual(missing("`libs/core/kernel/` `libs/core/kernel/container` `libs/core/kernel/go.mod:3` `.agents/rules/architecture.md#a-b`"), []);
  });

  it("reads a wildcard, a placeholder, a brace and an ellipsis as part of a path that needs one match", () => {
    assert.deepEqual(missing("`plugins/*/manifest.json` `plugins/<name>/manifest.json` `plugins/views-{tasks,files}` `libs/.../src` `plugins/(X)/manifest.json`"), []);
    assert.deepEqual(missing("`plugins/*/backend` `plugins/<name>/gone.json` `plugins/{a,b}` `plugins/(X)/gone`"), [
      "1 plugins/*/backend is in no file or folder of the tree",
      "1 plugins/<name>/gone.json is in no file or folder of the tree",
      "1 plugins/{a,b} is in no file or folder of the tree",
      "1 plugins/(X)/gone is in no file or folder of the tree",
    ]);
  });

  it("ignores text that is not a path of the repository", () => {
    const found = findMissingPaths([{ file: "AGENTS.md", text: "`make check_layers` `docs/gone.md` `src/gone.ts` `a/b` `libs gone/x`" }], TRACKED);
    assert.deepEqual(found.findings, []);
    assert.equal(found.read, 0);
  });
});
