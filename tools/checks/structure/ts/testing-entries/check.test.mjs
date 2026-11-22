import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findTestingInEntries, verifyNoTestingInEntries } from "./check.mjs";

const PACKAGE = JSON.stringify({ name: "sdk", exports: { ".": "./src/index.ts", "./testing": "./src/testing/index.ts", "./contracts": { import: "./src/contracts/index.ts" } } });

const find = (files) => findTestingInEntries(Object.keys(files), (file) => files[file]);

describe("no testing module is reachable from a non-testing entry", () => {
  it("reports a testing module re-exported by the main entry, with the chain", () => {
    const { findings, entries } = find({
      "libs/sdk/package.json": PACKAGE,
      "libs/sdk/src/index.ts": "export * from './plugin'\n",
      "libs/sdk/src/plugin.ts": "export { createMock } from './testing/index'\n",
      "libs/sdk/src/testing/index.ts": "export const createMock = 1\n",
      "libs/sdk/src/contracts/index.ts": "export const a = 1\n",
    });
    assert.equal(entries, 2);
    assert.deepEqual(findings, [
      {
        file: "libs/sdk/src/testing/index.ts",
        line: 1,
        symbol: "the entry . of libs/sdk reaches it through libs/sdk/src/index.ts > libs/sdk/src/plugin.ts > libs/sdk/src/testing/index.ts",
      },
    ]);
  });

  it("reports a testing module reached by a dynamic import or a nested entry", () => {
    const { findings } = find({
      "libs/sdk/package.json": PACKAGE,
      "libs/sdk/src/index.ts": "export const a = 1\n",
      "libs/sdk/src/testing/index.ts": "export const b = 1\n",
      "libs/sdk/src/contracts/index.ts": "export const load = () => import('./testing')\n",
      "libs/sdk/src/contracts/testing/index.ts": "export const c = 1\n",
    });
    assert.deepEqual(findings.map((one) => one.file), ["libs/sdk/src/contracts/testing/index.ts"]);
  });

  it("passes a testing entry that reaches testing modules and a package without them", () => {
    const { findings, entries } = find({
      "libs/sdk/package.json": PACKAGE,
      "libs/sdk/src/index.ts": "export const a = 1\n",
      "libs/sdk/src/contracts/index.ts": "export const b = 1\n",
      "libs/sdk/src/testing/index.ts": "export * from './mock'\n",
      "libs/sdk/src/testing/mock.ts": "export const c = 1\n",
    });
    assert.deepEqual(findings, []);
    assert.equal(entries, 2);
  });

  it("passes on the repository", () => {
    verifyNoTestingInEntries();
  });
});
