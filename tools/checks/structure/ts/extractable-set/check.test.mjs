import assert from "node:assert/strict";
import { test } from "node:test";
import { findExtractableViolations, findSharedLeafViolations } from "./check.mjs";

const pkg = (file, name, fields = {}) => ({ file, manifest: { name, ...fields } });

const TREE = [
  pkg("libs/shared/protocol/package.json", "@numen/protocol"),
  pkg("libs/core/client/package.json", "@numen/core-client"),
  pkg("libs/core/store/package.json", "@numen/core-store"),
  pkg("libs/plugin/sdk-ts/package.json", "@numen/plugin-sdk", { dependencies: { "@numen/ui": "*" }, devDependencies: { "@numen/protocol": "*" } }),
  pkg("libs/ui/components/package.json", "@numen/ui", { dependencies: { vue: "^3" } }),
  pkg("libs/ui/workbench/package.json", "@numen/workbench", { dependencies: { "@numen/ui": "*" } }),
  pkg("libs/shared/text/package.json", "@numen/text"),
  pkg("libs/shared/graph/package.json", "@numen/graph", { dependencies: { "@numen/text": "*", lodash: "^4" } }),
  pkg("plugins/a/package.json", "@numen/a", { dependencies: { "@numen/plugin-sdk": "workspace:*", "@numen/ui": "workspace:*", "@numen/text": "*" } }),
];

const symbols = (found) => found.map(({ file, symbol }) => `${file}: ${symbol}`);

test("a tree whose packages stay inside the set reports nothing", () => {
  assert.deepEqual(findSharedLeafViolations(TREE), []);
  assert.deepEqual(findExtractableViolations(TREE), []);
});

test("a shared package that declares the SDK, a UI package or a core package is reported in every field", () => {
  const found = findSharedLeafViolations([
    ...TREE,
    pkg("libs/shared/bad/package.json", "@numen/bad", {
      dependencies: { "@numen/plugin-sdk": "*" },
      devDependencies: { "@numen/core-store": "*" },
      peerDependencies: { "@numen/ui": "*" },
      optionalDependencies: { "@numen/core-client": "*" },
    }),
  ]);
  assert.equal(found.length, 4);
  assert.match(found[0].symbol, /@numen\/bad declares @numen\/plugin-sdk in dependencies \(libs\/plugin\/sdk-ts\)/);
  assert.match(found[1].symbol, /in devDependencies \(libs\/core\/store\)/);
  assert.match(found[2].symbol, /in peerDependencies/);
  assert.match(found[3].symbol, /in optionalDependencies/);
});

test("the leaf rule reads only libs/shared", () => {
  const loose = pkg("libs/plugin/sdk-ts/package.json", "@numen/plugin-sdk", { dependencies: { "@numen/core-store": "*" } });
  assert.deepEqual(findSharedLeafViolations([...TREE.slice(0, 3), loose]), []);
});

test("the SDK, a UI package and a plugin that declare a package outside the set are reported", () => {
  const found = findExtractableViolations([
    ...TREE,
    pkg("libs/plugin/sdk-ts/package.json", "@numen/plugin-sdk", { devDependencies: { "@numen/core-store": "*" }, peerDependencies: { "@numen/ui": "*" } }),
    pkg("libs/ui/workbench/package.json", "@numen/workbench", { dependencies: { "@numen/core-client": "*" } }),
    pkg("plugins/b/package.json", "@numen/b", { dependencies: { "@numen/a": "*", "@numen/core-store": "*" } }),
  ]);
  assert.deepEqual(
    symbols(found).map((line) => line.replace(/: an extractable.*/, "")),
    [
      "libs/plugin/sdk-ts/package.json: @numen/plugin-sdk declares @numen/core-store in devDependencies (libs/core/store)",
      "libs/ui/workbench/package.json: @numen/workbench declares @numen/core-client in dependencies (libs/core/client)",
      "plugins/b/package.json: @numen/b declares @numen/a in dependencies (plugins/a)",
      "plugins/b/package.json: @numen/b declares @numen/core-store in dependencies (libs/core/store)",
    ],
  );
});

test("a package named like a folder of the set but not below it is reported", () => {
  const lookalike = pkg("libs/sharedx/package.json", "@numen/sharedx");
  const found = findExtractableViolations([...TREE, lookalike, pkg("plugins/c/package.json", "@numen/c", { dependencies: { "@numen/sharedx": "*" } })]);
  assert.equal(found.length, 1);
});

test("third-party packages, including one whose name is not a package of the repository, pass", () => {
  const found = findExtractableViolations([...TREE, pkg("plugins/d/package.json", "@numen/d", { dependencies: { vue: "^3", "@numen/elsewhere": "*" } })]);
  assert.deepEqual(found, []);
});
