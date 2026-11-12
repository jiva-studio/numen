import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, before, describe, it } from "node:test";
import { findDependencyViolations, getAllowedPackages, listCruiseRuns, listPackages } from "./check.mjs";

const json = (value) => JSON.stringify(value);

/** A repository in the layout of this one: libraries, a window and two plugins, with one violation of each rule. */
const FILES = {
  "libs/core/protocol/package.json": json({ name: "@numen/protocol" }),
  "libs/core/protocol/src/index.ts": "export const message = 1\n",
  "libs/plugin/sdk-ts/package.json": json({ name: "@numen/plugin-sdk" }),
  "libs/plugin/sdk-ts/src/index.ts": [
    "import { slots } from './slots'",
    "import { menu } from './slots/menu'",
    "import { helper } from './testing/helper'",
    "export const sdk = [slots, menu, helper]",
    "",
  ].join("\n"),
  "libs/plugin/sdk-ts/src/slots/index.ts": "import { menu } from './menu'\nexport const slots = menu\n",
  "libs/plugin/sdk-ts/src/slots/menu.ts": "export const menu = 1\n",
  "libs/plugin/sdk-ts/src/commands/index.ts": "import { slots } from '../slots'\nexport const commands = slots\n",
  "libs/plugin/sdk-ts/src/commands/run.ts": "import { menu } from '../slots/menu'\nexport const run = menu\n",
  "libs/plugin/sdk-ts/src/testing/helper.ts": "import { run } from '../commands/run'\nexport const helper = run\n",
  "libs/ui/components/package.json": json({ name: "@numen/ui" }),
  "libs/ui/components/src/index.ts": "export const button = 1\n",
  "libs/mystery/package.json": json({ name: "@numen/mystery" }),
  "libs/mystery/src/index.ts": "export const mystery = 1\n",
  "plugins/alpha/package.json": json({ name: "@numen/alpha" }),
  "plugins/alpha/frontend/index.ts": [
    "import { sdk } from '@numen/plugin-sdk'",
    "import { message } from '@numen/protocol'",
    "import { beta } from '../../beta/frontend/index'",
    "export const alpha = [sdk, message, beta]",
    "",
  ].join("\n"),
  "plugins/beta/package.json": json({ name: "@numen/beta" }),
  "plugins/beta/frontend/index.ts": "import { button } from '@numen/ui'\nexport const beta = button\n",
  "apps/window/package.json": json({ name: "@numen/window" }),
  "apps/window/tsconfig.json": json({ compilerOptions: { baseUrl: ".", paths: { "@/*": ["./src/*"] } } }),
  "apps/window/src/first.ts": "import { second } from './second'\nexport const first = second\n",
  "apps/window/src/second.ts": "import { first } from '@/first'\nexport const second = () => first\n",
  "apps/window/src/typed.ts": "import type { Other } from './other'\nexport const typed: Other = 1\n",
  "apps/window/src/other.ts": "import { typed } from './typed'\nexport type Other = number\nexport const other = typed\n",
  "apps/window/src/plugin.ts": "import { sdk } from '@numen/plugin-sdk'\nexport const plugin = sdk\n",
  "apps/window/src/note/lib/count.ts": "import { view } from '../ui/view'\nexport const count = view\n",
  "apps/window/src/note/ui/view.ts": "export const view = 1\n",
};

describe("the dependency graph over a repository with a violation of each rule", () => {
  let root;
  let result;
  before(() => {
    root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "dependency-graph-")));
    for (const [name, text] of Object.entries(FILES)) {
      fs.mkdirSync(path.dirname(path.join(root, name)), { recursive: true });
      fs.writeFileSync(path.join(root, name), text);
    }
    result = findDependencyViolations(root);
  });
  after(() => fs.rmSync(root, { recursive: true, force: true }));

  const rulesOf = (file) => [...new Set(result.findings.filter((one) => one.file === file).map((one) => one.symbol.split(":")[0]))];

  it("fails a library that the package matrix has no row for", () => {
    assert.deepEqual(
      result.findings.filter((one) => one.symbol.startsWith("library ")).map((one) => `${one.file} ${one.symbol}`),
      ["libs/mystery/package.json library @numen/mystery has no row in the package matrix"],
    );
  });

  it("runs the window whose tsconfig declares paths on its own", () => {
    const runs = listCruiseRuns(root);
    assert.deepEqual(runs.find((run) => run.tsConfig)?.roots, ["apps/window/src"]);
    assert.ok(result.cruised >= Object.keys(FILES).length / 2);
  });

  it("reports a plugin that imports the schema and a file of another plugin", () => {
    assert.deepEqual(rulesOf("plugins/alpha/frontend/index.ts").sort(), ["plugins-alpha-reaches-only-its-packages", "plugins-alpha-stays-in-its-directory"]);
    assert.deepEqual(rulesOf("plugins/beta/frontend/index.ts"), []);
  });

  it("reports a window that imports the plugin SDK", () => {
    assert.deepEqual(rulesOf("apps/window/src/plugin.ts"), ["apps-window-reaches-only-its-packages"]);
  });

  it("reports a runtime cycle closed through the alias, and not one only a type import closes", () => {
    assert.ok(rulesOf("apps/window/src/first.ts").includes("no-circular"));
    assert.deepEqual(rulesOf("apps/window/src/typed.ts"), []);
    assert.deepEqual(rulesOf("apps/window/src/other.ts"), []);
  });

  it("reports a lib segment that imports the ui segment of its slice", () => {
    assert.deepEqual(rulesOf("apps/window/src/note/lib/count.ts"), ["segment-lib-imports-no-higher-segment"]);
  });

  it("reports a file that imports a folder of its flat library by a deeper file than the index", () => {
    assert.deepEqual(rulesOf("libs/plugin/sdk-ts/src/index.ts"), ["area-imported-through-its-index"]);
    assert.deepEqual(rulesOf("libs/plugin/sdk-ts/src/commands/run.ts"), ["area-imported-through-its-index"]);
    assert.deepEqual(rulesOf("libs/plugin/sdk-ts/src/testing/helper.ts"), ["area-imported-through-its-index"]);
    assert.deepEqual(rulesOf("libs/plugin/sdk-ts/src/commands/index.ts"), []);
    assert.deepEqual(rulesOf("libs/plugin/sdk-ts/src/slots/index.ts"), []);
  });
});

describe("the packages a package may depend on", () => {
  const packages = [
    { dir: "libs/core/protocol", name: "@numen/protocol", kind: "library" },
    { dir: "libs/plugin/sdk-ts", name: "@numen/plugin-sdk", kind: "library" },
    { dir: "libs/core/client", name: "@numen/core-client", kind: "library" },
    { dir: "apps/window", name: "@numen/window", kind: "application" },
    { dir: "plugins/alpha", name: "@numen/alpha", kind: "plugin" },
  ];
  it("gives a plugin the SDK and the UI, a window every library but the SDK and the schema, a library its row", () => {
    assert.deepEqual(getAllowedPackages(packages[4], packages), ["@numen/plugin-sdk", "@numen/ui"]);
    assert.deepEqual(getAllowedPackages(packages[3], packages), ["@numen/core-client"]);
    assert.deepEqual(getAllowedPackages(packages[2], packages, { "@numen/core-client": ["@numen/protocol"] }), ["@numen/protocol"]);
    assert.equal(getAllowedPackages(packages[2], packages, {}), null);
  });
  it("reads the packages of this repository", () => {
    const kinds = new Set(listPackages(path.resolve(import.meta.dirname, "../../../../..")).map((one) => one.kind));
    assert.deepEqual([...kinds].sort(), ["application", "library", "plugin"]);
  });
});
