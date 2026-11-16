import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { runTestOnlyModulesCheck } from "./check.mjs";

const REPO = path.resolve(import.meta.dirname, "../../../../..");
const roots = [];
after(() => roots.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

/** A package with an entry, a module the entry imports, a test of it and the given extra sources, linked to the installed knip. */
function createRepo(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "test-only-modules-"));
  roots.push(dir);
  const all = {
    "package.json": JSON.stringify({ name: "fixture", private: true, type: "module" }),
    "knip.json": JSON.stringify({ entry: ["src/index.ts!", "src/**/*.test.ts", "src/**/*.stories.ts"], project: ["src/**/*.ts!"] }),
    "src/index.ts": "import { formatWords } from './words'\n\nexport const label = formatWords(2)\n",
    "src/words.ts": "export function formatWords(count: number): string {\n  return String(count)\n}\n",
    "src/words.test.ts": "import { formatWords } from './words'\n\nexport const text = formatWords(1)\n",
    ...files,
  };
  for (const [name, text] of Object.entries(all)) {
    fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
    fs.writeFileSync(path.join(dir, name), text);
  }
  fs.symlinkSync(path.join(REPO, "node_modules"), path.join(dir, "node_modules"));
  return dir;
}

describe("the test-only modules check", () => {
  it("reports a module that only a test imports, and not the test", async () => {
    const repo = createRepo({
      "src/helper.ts": "export const helper = 1\n",
      "src/helper.test.ts": "import { helper } from './helper'\n\nexport const used = helper\n",
    });
    const result = await runTestOnlyModulesCheck({ repoRoot: repo });
    assert.equal(result.pass, false);
    assert.match(result.details, /src\/helper\.ts/);
    assert.doesNotMatch(result.details, /\.test\.ts/);
    assert.doesNotMatch(result.details, /words\.ts/);
  });

  it("reports a module that only a story imports, and not the story", async () => {
    const repo = createRepo({
      "src/sample.ts": "export const sample = 1\n",
      "src/words.stories.ts": "import { sample } from './sample'\n\nexport default { title: String(sample) }\n",
    });
    const result = await runTestOnlyModulesCheck({ repoRoot: repo });
    assert.equal(result.pass, false);
    assert.match(result.details, /src\/sample\.ts/);
    assert.doesNotMatch(result.details, /stories/);
  });

  it("passes when every module has a shipped importer", async () => {
    const result = await runTestOnlyModulesCheck({ repoRoot: createRepo({}) });
    assert.equal(result.pass, true, result.details);
  });
});
