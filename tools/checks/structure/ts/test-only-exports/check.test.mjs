import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { runTestOnlyExportsCheck } from "./check.mjs";

const REPO = path.resolve(import.meta.dirname, "../../../../..");
const roots = [];
after(() => roots.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

/** A package of the given sources with an entry and its tests, linked to the installed knip. */
function createRepo(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "test-only-exports-"));
  roots.push(dir);
  const all = {
    "package.json": JSON.stringify({ name: "fixture", private: true, type: "module" }),
    "knip.json": JSON.stringify({ entry: ["src/index.ts!", "src/**/*.test.ts"], project: ["src/**/*.ts!"] }),
    "src/index.ts": "import { formatWords } from './words'\n\nexport const label = formatWords(2)\n",
    ...files,
  };
  for (const [name, text] of Object.entries(all)) {
    fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
    fs.writeFileSync(path.join(dir, name), text);
  }
  fs.symlinkSync(path.join(REPO, "node_modules"), path.join(dir, "node_modules"));
  return dir;
}

const WORDS = "export function formatWords(count: number): string {\n  return String(count)\n}\n\nexport function countWords(text: string): number {\n  return text.split(' ').length\n}\n";

describe("the test-only exports check", () => {
  it("reports an export that only a test imports", async () => {
    const repo = createRepo({
      "src/words.ts": WORDS,
      "src/words.test.ts": "import { countWords } from './words'\n\nexport const counted = countWords('one two')\n",
    });
    const result = await runTestOnlyExportsCheck({ repoRoot: repo });
    assert.equal(result.pass, false);
    assert.match(result.details, /countWords/);
    assert.doesNotMatch(result.details, /formatWords/);
  });

  it("passes when every export has a shipped user", async () => {
    const repo = createRepo({
      "src/words.ts": WORDS,
      "src/index.ts": "import { countWords, formatWords } from './words'\n\nexport const label = formatWords(countWords('one two'))\n",
      "src/words.test.ts": "import { countWords } from './words'\n\nexport const counted = countWords('one two')\n",
    });
    const result = await runTestOnlyExportsCheck({ repoRoot: repo });
    assert.equal(result.pass, true, result.details);
  });
});
