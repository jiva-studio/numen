import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findTsPackagesWithoutEslint } from "./check.mjs";
describe("the TypeScript packages without a lint configuration", () => {
  it("finds a package that has no eslint.config.js", () => {
    const found = findTsPackagesWithoutEslint(["libs/a", "apps/b"], (dir) => dir === "libs/a");
    assert.deepEqual(found.map((one) => one.file), ["apps/b/tsconfig.json"]);
  });
});
