#!/usr/bin/env node
/**
 * Runs the checks of the registry (`rules.json`) and prints one report.
 *
 *   node run.mjs                        every check
 *   node run.mjs --tags fast,ts         the checks that carry all the tags (and)
 *   node run.mjs --tags go|ts           the checks that carry one of the groups (or); `,` binds tighter
 *   node run.mjs --tags ?               the tags and what they mean
 *   node run.mjs --exclude slow         without the checks that carry the tag
 *   node run.mjs --only name,name       the named checks
 *   node run.mjs --changed              the checks whose paths a change of the branch touches, and the
 *                                       planted-violation tests of the checks whose own code it touches
 *   node run.mjs --list                 what runs and why; nothing runs
 *   node run.mjs --selftest             the planted-violation tests of the selected checks
 *   node run.mjs --without-lint         skip the verifiers that run a package's own linter
 *   node run.mjs --without-go-tests     skip the checks that run as Go tests
 */
import { spawn } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import process from "node:process";
import { pathToFileURL } from "node:url";
import { isMatching, listChangedFiles } from "./lib/changed.mjs";
import { printReport } from "./lib/report.mjs";

const CHECKS_DIR = import.meta.dirname;
const REPO_ROOT = process.env.NUMEN_CHECKS_ROOT ? path.resolve(process.env.NUMEN_CHECKS_ROOT) : path.resolve(CHECKS_DIR, "../..");
const GO_DIR = path.join(CHECKS_DIR, "structure/go");
const CHECKS_REL = path.relative(REPO_ROOT, CHECKS_DIR).split(path.sep).join("/");

/** The registry: the tag vocabulary and the checks. */
export function readRegistry() {
  return JSON.parse(fs.readFileSync(path.join(CHECKS_DIR, "rules.json"), "utf8"));
}

/** Whether a check carries the tags of an expression: groups joined by `|`, tags joined by `,` inside a group. */
export function isTagged(entry, expression) {
  return expression.split("|").some((group) => group.split(",").map((t) => t.trim()).filter(Boolean).every((tag) => entry.tags.includes(tag)));
}

function parseArgs(argv) {
  const options = { tags: "", exclude: "", only: "", changed: false, list: false, selftest: false, withoutLint: false, withoutGoTests: false, worker: "", result: "" };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--tags") options.tags = argv[++i] ?? "";
    else if (arg === "--exclude") options.exclude = argv[++i] ?? "";
    else if (arg === "--only") options.only = argv[++i] ?? "";
    else if (arg === "--changed") options.changed = true;
    else if (arg === "--list") options.list = true;
    else if (arg === "--selftest") options.selftest = true;
    else if (arg === "--without-lint") options.withoutLint = true;
    else if (arg === "--without-go-tests") options.withoutGoTests = true;
    else if (arg === "--worker") options.worker = argv[++i] ?? "";
    else if (arg === "--result") options.result = argv[++i] ?? "";
    else {
      console.error(`Unknown option: ${arg}`);
      process.exit(2);
    }
  }
  return options;
}

/** The folders and files whose change selects a check beside its paths: its own code and the shared library. */
export function getOwnPaths(entry) {
  const own = [`${CHECKS_REL}/lib/**`];
  if (entry.runner === "go") own.push(`${CHECKS_REL}/structure/go/${entry.target}/**`, `${CHECKS_REL}/structure/go/load/**`, `${CHECKS_REL}/structure/go/go.mod`);
  if (entry.runner === "verifiers") own.push(`${CHECKS_REL}/${path.dirname(entry.module)}/**`);
  if (entry.runner === "command") {
    own.push(`${path.posix.join(CHECKS_REL, path.posix.dirname(entry.selftest))}/**`);
    own.push(...entry.cmd.slice(1).filter((arg) => /\.m?js$/.test(arg) && fs.existsSync(path.join(REPO_ROOT, arg))));
  }
  if (entry.selftest) own.push(path.posix.join(CHECKS_REL, entry.selftest));
  return own;
}

/**
 * The command of a check's planted-violation tests: every test file of its folder. Null when the check itself runs
 * those tests: a Go check, a rule test, the registry.
 */
function getSelftestCmd(entry) {
  if (entry.selftestCmd) return entry.selftestCmd;
  if (entry.runner === "go" || entry.runner === "eslint" || entry.runner === "stylelint") return null;
  if (entry.cmd?.some((arg) => arg.endsWith(entry.selftest))) return null;
  const dir = path.join(CHECKS_DIR, path.dirname(entry.selftest));
  const tests = fs.readdirSync(dir).filter((name) => name.endsWith(".test.mjs")).sort().map((name) => path.join(dir, name));
  return ["node", "--test", ...tests];
}

/** Decides which checks run, and why each one does or does not. */
export function selectChecks(registry, options, changed) {
  const wholeRunner = changed?.some((file) => isMatching(file, [`${CHECKS_REL}/run.mjs`, `${CHECKS_REL}/rules.json`, `${CHECKS_REL}/lib/*.mjs`]));
  const only = options.only ? new Set(options.only.split(",").map((s) => s.trim())) : null;
  return registry.checks.map((entry) => {
    let reason = "all";
    if (only && !only.has(entry.name)) return { entry, run: false, reason: "not named" };
    if (only) reason = "named";
    if (options.tags && !isTagged(entry, options.tags)) return { entry, run: false, reason: `lacks tags ${options.tags}` };
    if (options.withoutGoTests && entry.runner === "go") return { entry, run: false, reason: "go tests are left out" };
    if (options.exclude && options.exclude.split(",").some((tag) => entry.tags.includes(tag.trim()))) return { entry, run: false, reason: `excluded by tag ${options.exclude}` };
    if (changed) {
      const own = changed.find((file) => isMatching(file, getOwnPaths(entry)));
      const touched = own ?? changed.find((file) => isMatching(file, entry.paths));
      if (touched) reason = `changed ${touched}`;
      else if (wholeRunner) reason = "the runner or the registry changed";
      else return { entry, run: false, reason: "no path changed" };
      const selftest = Boolean(own || wholeRunner) && getSelftestCmd(entry) !== null;
      return { entry, run: true, reason: selftest ? `${reason}, with its selftest` : reason, selftest };
    }
    return { entry, run: true, reason, selftest: false };
  });
}

function runGo(entries) {
  return new Promise((resolve) => {
    const packages = entries.map((e) => `./${e.target}`);
    const child = spawn("go", ["test", "-p", "3", "-count=1", ...packages], { cwd: GO_DIR, env: process.env });
    let text = "";
    child.stdout.on("data", (d) => (text += d));
    child.stderr.on("data", (d) => (text += d));
    const started = Date.now();
    child.on("close", (code) => {
      const results = new Map();
      let buffer = [];
      for (const line of text.split("\n")) {
        const status = /^(ok|FAIL|\?)\s+\S+\/structure\/go\/(\S+)\s/.exec(line);
        if (status) {
          results.set(status[2], { ok: status[1] !== "FAIL", output: buffer.join("\n") });
          buffer = [];
        } else buffer.push(line);
      }
      resolve(
        entries.map((e) => {
          const one = results.get(e.target);
          if (one) return { name: e.name, status: one.ok ? "pass" : "fail", ms: Date.now() - started, output: one.output };
          return { name: e.name, status: code === 0 ? "pass" : "fail", ms: Date.now() - started, output: code === 0 ? "" : text };
        }),
      );
    });
    child.on("error", (err) => resolve(entries.map((e) => ({ name: e.name, status: "fail", ms: 0, output: String(err) }))));
  });
}

async function runVerifiers(entry, options) {
  const started = Date.now();
  const loaded = await import(pathToFileURL(path.join(CHECKS_DIR, entry.module)).href);
  const verifiers = loaded[entry.export].filter((v) => !(options.withoutLint && v.isLint));
  const lines = [];
  let failed = false;
  if (verifiers.some((v) => v.tools?.includes("golangci-lint"))) {
    try {
      (await import("./lib/golangci.mjs")).assertGolangciLintBuild();
    } catch (err) {
      return { name: entry.name, status: "fail", ms: Date.now() - started, output: "Tool version check failed: " + err.message };
    }
  }
  for (const verifier of verifiers) {
    try {
      await verifier.run();
    } catch (err) {
      failed = true;
      lines.push(`  ${verifier.name} [${verifier.id}]`);
      if (err.stdout) lines.push(err.stdout.toString());
      if (err.stderr) lines.push(err.stderr.toString());
      if (!err.stdout && !err.stderr && err.message) lines.push("    " + err.message);
    }
  }
  return { name: entry.name, status: failed ? "fail" : "pass", ms: Date.now() - started, output: lines.join("\n") };
}


function runCommand(name, cmd, cwd = REPO_ROOT) {
  return new Promise((resolve) => {
    const started = Date.now();
    const child = spawn(cmd[0], cmd.slice(1), { cwd, env: process.env });
    let text = "";
    child.stdout.on("data", (d) => (text += d));
    child.stderr.on("data", (d) => (text += d));
    child.on("error", (err) => resolve({ name, status: "fail", ms: Date.now() - started, output: text + String(err) }));
    child.on("close", (code) => resolve({ name, status: code === 0 ? "pass" : "fail", ms: Date.now() - started, output: text }));
  });
}

/** Runs the verifiers of a check in a process of its own, so that checks run side by side. */
async function runVerifiersApart(entry, options) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "numen-check-"));
  const file = path.join(dir, "result.json");
  const args = [import.meta.filename, "--worker", entry.name, "--result", file, ...(options.withoutLint ? ["--without-lint"] : [])];
  const run = await runCommand(entry.name, [process.execPath, ...args], process.cwd());
  let result;
  try {
    result = JSON.parse(fs.readFileSync(file, "utf8"));
  } catch {
    result = { status: "fail", output: run.output || "the check stopped without a result" };
    run.output = "";
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
  const output = [run.output.trimEnd(), result.output].filter(Boolean).join("\n");
  return { name: entry.name, status: result.status, ms: run.ms, output };
}

/** Calls the tasks, at most `limit` at once, and returns their results in the order of the tasks. */
async function runPool(tasks, limit) {
  const results = new Array(tasks.length);
  let next = 0;
  const work = async () => {
    while (next < tasks.length) {
      const index = next++;
      results[index] = await tasks[index]();
    }
  };
  await Promise.all(Array.from({ length: Math.min(limit, tasks.length) }, work));
  return results;
}

function runCheck(entry, options) {
  if (entry.runner === "verifiers") return runVerifiersApart(entry, options);
  if (entry.runner === "command") return runCommand(entry.name, entry.cmd);
  if (entry.runner === "eslint" || entry.runner === "stylelint") return runCommand(entry.name, ["node", "--test", path.join(CHECKS_DIR, entry.selftest)]);
  return Promise.resolve({ name: entry.name, status: "fail", ms: 0, output: `unknown runner ${entry.runner}` });
}

/**
 * Runs the selected checks side by side and returns their results, the Go checks first. A check named in
 * `withSelftest` is followed by its planted-violation tests.
 */
export async function runChecks(entries, options, withSelftest = new Set()) {
  const goEntries = entries.filter((e) => e.runner === "go");
  const goDone = goEntries.length > 0 ? runGo(goEntries) : Promise.resolve([]);
  const tasks = [];
  for (const entry of entries.filter((e) => e.runner !== "go")) {
    if (options.selftest) {
      tasks.push(() => runCommand(entry.name, getSelftestCmd(entry) ?? ["node", "--test", path.join(CHECKS_DIR, entry.selftest)]));
      continue;
    }
    tasks.push(() => runCheck(entry, options));
    if (withSelftest.has(entry.name)) tasks.push(() => runCommand(`${entry.name} (selftest)`, getSelftestCmd(entry)));
  }
  const done = await runPool(tasks, Math.max(1, os.availableParallelism()));
  return [...(await goDone), ...done];
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const registry = readRegistry();
  if (options.worker) {
    const entry = registry.checks.find((e) => e.name === options.worker);
    fs.writeFileSync(options.result, JSON.stringify(await runVerifiers(entry, options)));
    return 0;
  }
  if (options.tags === "?") {
    for (const [tag, description] of Object.entries(registry.tags)) console.log(`${tag.padEnd(10)} ${description}`);
    return 0;
  }
  let changed = null;
  if (options.changed) {
    try {
      changed = listChangedFiles(REPO_ROOT);
    } catch (err) {
      console.error(`Cannot select the changed checks: ${err.message}`);
      return 2;
    }
  }
  const selection = selectChecks(registry, options, changed);
  if (options.list) {
    for (const one of selection) console.log(`${one.run ? "run " : "skip"} ${one.entry.name.padEnd(46)} ${one.reason}`);
    console.log(`\n${selection.filter((s) => s.run).length} of ${selection.length} checks run`);
    return 0;
  }
  const entries = selection.filter((s) => s.run).map((s) => s.entry);
  if (entries.length === 0) {
    console.log("No check selected.");
    return 0;
  }
  const withSelftest = new Set(options.selftest ? [] : selection.filter((s) => s.run && s.selftest).map((s) => s.entry.name));
  const besides = withSelftest.size > 0 ? `, ${withSelftest.size} with their planted-violation tests` : "";
  console.log(`Running ${entries.length} of ${selection.length} checks${options.selftest ? " (planted-violation tests)" : besides}\n`);
  // The registry test then compares the verifiers of the checks the change touches, not all of them.
  if (options.changed) process.env.NUMEN_CHECKS_CHANGED = "1";
  return printReport(await runChecks(entries, options, withSelftest));
}

if (process.argv[1] && path.resolve(process.argv[1]) === import.meta.filename) process.exit(await main());
