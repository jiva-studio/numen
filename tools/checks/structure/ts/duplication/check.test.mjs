import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { runDuplicationCheck } from "./check.mjs";

const REPO = path.resolve(import.meta.dirname, "../../../../..");
const roots = [];
after(() => roots.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

/** A repository of two library files, linked to the installed jscpd. */
function createRepo(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "duplication-"));
  roots.push(dir);
  fs.mkdirSync(path.join(dir, "plugins"));
  fs.mkdirSync(path.join(dir, "apps"));
  for (const [name, text] of Object.entries(files)) {
    fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
    fs.writeFileSync(path.join(dir, name), text);
  }
  fs.symlinkSync(path.join(REPO, "node_modules"), path.join(dir, "node_modules"));
  return dir;
}

/** A block of code of `count` distinct statements, named after `seed`. */
function createBlock(seed, count) {
  return Array.from({ length: count }, (_, i) => `export function ${seed}Step${i}(input: number): number {\n  const scaled = input * ${i + 3};\n  return scaled + ${i * 7} - input / ${i + 2};\n}\n`).join("\n");
}

describe("the duplication check", () => {
  it("reports a file copied into another", async () => {
    const block = createBlock("copy", 40);
    const repo = createRepo({ "libs/a/one.ts": block, "libs/a/two.ts": block });
    const result = await runDuplicationCheck({ repoRoot: repo });
    assert.equal(result.pass, false);
    assert.match(result.details, /host: .*exceeds threshold/);
  });

  it("passes two files that share nothing", async () => {
    const repo = createRepo({ "libs/a/one.ts": createBlock("first", 40), "libs/a/two.ts": createBlock("second", 40).replaceAll("scaled", "other").replaceAll("input", "value") });
    const result = await runDuplicationCheck({ repoRoot: repo, threshold: 50 });
    assert.equal(result.pass, true);
  });
});
