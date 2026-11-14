import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { describe, it, after } from "node:test";
import { isDeepStrictEqual } from "node:util";
import { getPathPattern, isMatching, listChangedFiles, readBaseFile } from "./lib/changed.mjs";
import { listTrackedFiles, plantTree } from "./lib/plant.mjs";
import { getOwnPaths, isTagged, readRegistry, selectChecks } from "./run.mjs";

const CHECKS_DIR = import.meta.dirname;
const REPO_ROOT = path.resolve(CHECKS_DIR, "../..");
const registry = readRegistry();
const tracked = listTrackedFiles(REPO_ROOT).filter((file) => fs.existsSync(path.join(REPO_ROOT, file)));

describe("the registry", () => {
  it("describes every tag of its vocabulary", () => {
    for (const [tag, description] of Object.entries(registry.tags)) {
      assert.ok(typeof description === "string" && description.length > 10, `the tag ${tag} has no description`);
    }
  });

  it("gives each check a unique name, paths, a runner and tags of the vocabulary", () => {
    const names = new Set();
    for (const entry of registry.checks) {
      assert.ok(entry.name && !names.has(entry.name), `the name ${entry.name} is empty or used twice`);
      names.add(entry.name);
      assert.ok(Array.isArray(entry.paths) && entry.paths.length > 0, `${entry.name} lists no paths`);
      assert.ok(Array.isArray(entry.tags) && entry.tags.length > 0, `${entry.name} has no tags`);
      for (const tag of entry.tags) assert.ok(tag in registry.tags, `${entry.name} has the tag ${tag}, which the vocabulary does not declare`);
    }
  });

  it("tags each check fast or slow, and not both", () => {
    for (const entry of registry.checks) {
      const speeds = entry.tags.filter((tag) => tag === "fast" || tag === "slow");
      assert.equal(speeds.length, 1, `${entry.name} must carry exactly one of fast and slow, has ${speeds.length}`);
    }
  });

  it("points each check at a planted-violation test that exists", () => {
    for (const entry of registry.checks) {
      assert.ok(entry.selftest, `${entry.name} names no selftest`);
      assert.ok(fs.existsSync(path.join(CHECKS_DIR, entry.selftest)), `${entry.name}: the selftest ${entry.selftest} does not exist`);
    }
  });

  it("gives each runner what it needs", () => {
    for (const entry of registry.checks) {
      if (entry.runner === "go") assert.ok(fs.existsSync(path.join(CHECKS_DIR, "structure/go", entry.target, "check_test.go")), `${entry.name}: no Go test in ${entry.target}`);
      else if (entry.runner === "verifiers") {
        assert.ok(fs.existsSync(path.join(CHECKS_DIR, entry.module)), `${entry.name}: no module ${entry.module}`);
        assert.match(fs.readFileSync(path.join(CHECKS_DIR, entry.module), "utf8"), new RegExp(`export const ${entry.export}\\b`), `${entry.name}: ${entry.module} does not export ${entry.export}`);
      } else if (entry.runner === "command") assert.ok(Array.isArray(entry.cmd), `${entry.name} has no cmd`);
      else assert.ok(["eslint", "stylelint"].includes(entry.runner), `${entry.name}: unknown runner ${entry.runner}`);
    }
  });

  it("lists every check folder", () => {
    const listed = new Set(registry.checks.flatMap((e) => [e.runner === "go" ? `structure/go/${e.target}` : null, e.module ? path.dirname(e.module) : null, e.runner === "command" ? path.dirname(e.selftest) : null]).filter(Boolean));
    const folders = [];
    const goDir = path.join(CHECKS_DIR, "structure/go");
    for (const one of fs.readdirSync(goDir, { withFileTypes: true })) if (one.isDirectory() && one.name !== "load") folders.push(`structure/go/${one.name}`);
    for (const area of ["structure/ts", "docs", "style"]) {
      for (const one of fs.readdirSync(path.join(CHECKS_DIR, area), { withFileTypes: true })) {
        if (one.isDirectory() && fs.existsSync(path.join(CHECKS_DIR, area, one.name, "check.mjs"))) folders.push(`${area}/${one.name}`);
      }
    }
    assert.ok(folders.length > 0);
    for (const folder of folders) assert.ok(listed.has(folder), `${folder} holds a check and the registry does not list it`);
  });

  it("lists only paths that match a tracked file", () => {
    for (const entry of registry.checks) {
      for (const pattern of entry.paths) {
        const regex = getPathPattern(pattern);
        assert.ok(tracked.some((file) => regex.test(file)), `${entry.name}: the path ${pattern} matches no tracked file`);
      }
    }
  });

  it("selects by tags with and, or and exclusion", () => {
    const entry = { tags: ["structure", "go", "fast"] };
    assert.ok(isTagged(entry, "fast,go"));
    assert.ok(!isTagged(entry, "fast,ts"));
    assert.ok(isTagged(entry, "ts|go"));
    assert.ok(!isTagged(entry, "ts|window"));
  });
});

describe("the selection of changed checks", () => {
  const options = { tags: "", exclude: "", only: "", changed: true, list: false, selftest: false, withoutLint: false, withoutGoTests: false };
  const getRunningChecks = (changed) => selectChecks(registry, options, changed).filter((one) => one.run).map((one) => one.entry.name);

  it("selects no Go check for a change outside every Go path", () => {
    const goNames = new Set(registry.checks.filter((entry) => entry.runner === "go").map((entry) => entry.name));
    assert.deepEqual(getRunningChecks(["unrelated/none.bin"]).filter((name) => goNames.has(name)), []);
  });

  it("selects the kernel checks for a kernel file and not for a plugin file", () => {
    assert.ok(getRunningChecks(["libs/core/kernel/usecase/note/note.go"]).includes("go/layers"));
    assert.ok(!getRunningChecks(["plugins/views-flashcards-studio/backend/main.go"]).includes("go/layers"));
  });

  it("selects the checks that read the note library and the plugins for a change there", () => {
    const noteDomain = getRunningChecks(["libs/note/domain/connection.go"]);
    assert.ok(noteDomain.includes("go/transports"));
    assert.ok(noteDomain.includes("go/use-case-fields"));
    assert.ok(getRunningChecks(["plugins/engines-transcription/backend/main.go"]).includes("go/use-case-fields"));
    assert.ok(getRunningChecks(["plugins/readers-pdf/frontend/api/types.ts"]).includes("ts/api-types"));
  });

  it("selects the check whose own code changed, with the reason", () => {
    const one = selectChecks(registry, options, ["tools/checks/structure/go/layers/check_test.go"]).find((s) => s.entry.name === "go/layers");
    assert.ok(one.run);
    assert.match(one.reason, /^changed tools\/checks\/structure\/go\/layers\//);
  });

  it("adds the selftest of a check whose own code changed, and not of one whose paths changed", () => {
    const find = (changed, name) => selectChecks(registry, options, changed).find((s) => s.entry.name === name);
    assert.ok(find(["tools/checks/structure/ts/plugins/check.mjs"], "ts/plugins").selftest);
    assert.ok(find(["tools/checks/style/comments/check.mjs"], "style/comments").selftest);
    const byPath = find(["plugins/readers-pdf/frontend/View.vue"], "ts/plugins");
    assert.ok(byPath.run && !byPath.selftest);
  });

  it("runs no separate selftest for a check that is its own test", () => {
    for (const name of ["registry", "eslint/layers", "go/layers"]) {
      const one = selectChecks(registry, options, ["tools/checks/lib/changed.mjs"]).find((s) => s.entry.name === name);
      assert.ok(one.run && !one.selftest, name);
    }
  });

  it("selects every check when the runner or the registry changed", () => {
    assert.equal(getRunningChecks(["tools/checks/rules.json"]).length, registry.checks.length);
  });

  it("selects every check when no changed-file list is given", () => {
    const all = selectChecks(registry, { ...options, changed: false }, null);
    assert.ok(all.every((one) => one.run));
  });
});

/** The text of a run with the root of the tree it ran in replaced, so two trees can be compared. */
function normalize(text, root) {
  return text.split("/private" + root).join(root).split(root).join("<root>").split(REPO_ROOT).join("<root>").replace(/(?:\.\.\/)+(?:private)?<root>/g, "<root>").replace(/\(\d+(\.\d+)?s\)/g, "(time)");
}

function run(name, root) {
  return new Promise((resolve) => {
    const child = spawn("node", [path.join(CHECKS_DIR, "run.mjs"), "--only", name], { env: { ...process.env, NUMEN_CHECKS_ROOT: root } });
    let text = "";
    child.stdout.on("data", (d) => (text += d));
    child.stderr.on("data", (d) => (text += d));
    child.on("close", (status) => resolve({ status, text: normalize(text, root) }));
  });
}

// A check that reads a file outside its `paths` gives another answer on a tree that holds only the
// `paths`. The Go checks (they type-check the workspace through `go list`), the commands and the
// rule tests are not run this way: a reduced tree has no go.work to load.
function listComparedVerifiers() {
  const verifiers = registry.checks.filter((e) => e.runner === "verifiers");
  if (!process.env.NUMEN_CHECKS_CHANGED) return verifiers;
  const changed = listChangedFiles(REPO_ROOT);
  if (changed.some((file) => isMatching(file, ["tools/checks/run.mjs", "tools/checks/lib/**"]))) return verifiers;
  const baseText = readBaseFile(REPO_ROOT, "tools/checks/rules.json");
  const baseChecks = baseText ? JSON.parse(baseText).checks : [];
  return verifiers.filter((entry) => {
    const before = baseChecks.find((one) => one.name === entry.name);
    return !isDeepStrictEqual(entry, before) || changed.some((file) => isMatching(file, getOwnPaths(entry)));
  });
}

describe("the paths of the verifiers", { concurrency: 4 }, () => {
  const planted = [];
  after(() => planted.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));
  for (const entry of listComparedVerifiers()) {
    it(`${entry.name} gives the same answer on a tree reduced to its paths`, async () => {
      const reduced = plantTree(REPO_ROOT, entry.paths);
      planted.push(reduced);
      const [full, small] = await Promise.all([run(entry.name, REPO_ROOT), run(entry.name, reduced)]);
      assert.equal(small.status, full.status, `${entry.name}: the status differs on the reduced tree; it prints:\n${small.text.slice(0, 3000)}`);
      assert.equal(small.text.replace(/\(\d+\.\d+s\)/g, ""), full.text.replace(/\(\d+\.\d+s\)/g, ""), `${entry.name} reads a file outside its paths`);
    });
  }
});

