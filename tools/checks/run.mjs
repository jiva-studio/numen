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
 *   node run.mjs --changed              the checks whose paths a change of the branch touches
 *   node run.mjs --list                 what runs and why; nothing runs
 *   node run.mjs --selftest             the planted-violation tests of the selected checks
 *   node run.mjs --without-lint         skip the verifiers that run a package's own linter
 *   node run.mjs --without-go-tests     skip the checks that run as Go tests
 */
import { spawn, spawnSync } from "node:child_process";
import fs from "node:fs";
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
  const options = { tags: "", exclude: "", only: "", changed: false, list: false, selftest: false, withoutLint: false, withoutGoTests: false };
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
    else {
      console.error(`Unknown option: ${arg}`);
      process.exit(2);
    }
  }
  return options;
}

/** The folders and files whose change selects a check beside its paths: its own code and the shared library. */
function getOwnPaths(entry) {
  const own = [`${CHECKS_REL}/lib/**`];
  if (entry.runner === "go") own.push(`${CHECKS_REL}/structure/go/${entry.target}/**`, `${CHECKS_REL}/structure/go/load/**`, `${CHECKS_REL}/structure/go/go.mod`);
  if (entry.runner === "verifiers") own.push(`${CHECKS_REL}/${path.dirname(entry.module)}/**`);
  if (entry.selftest) own.push(path.posix.join(CHECKS_REL, entry.selftest));
  return own;
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
      const touched = changed.find((file) => isMatching(file, [...entry.paths, ...getOwnPaths(entry)]));
      if (touched) reason = `changed ${touched}`;
      else if (wholeRunner) reason = "the runner or the registry changed";
      else return { entry, run: false, reason: "no path changed" };
    }
    return { entry, run: true, reason };
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

function runCommand(entry, cmd, cwd = REPO_ROOT) {
  const started = Date.now();
  const run = spawnSync(cmd[0], cmd.slice(1), { cwd, encoding: "utf8", maxBuffer: 1 << 28, env: process.env });
  return { name: entry.name, status: run.status === 0 ? "pass" : "fail", ms: Date.now() - started, output: (run.stdout || "") + (run.stderr || "") + (run.error ? String(run.error) : "") };
}

/** Runs the selected checks and returns their results. */
export async function runChecks(entries, options) {
  const results = new Map();
  const goEntries = entries.filter((e) => e.runner === "go");
  const goDone = !options.selftest && goEntries.length > 0 ? runGo(goEntries) : Promise.resolve([]);
  if (options.selftest) {
    const tests = entries.filter((e) => e.runner !== "go");
    for (const entry of tests) results.set(entry.name, runCommand(entry, entry.selftestCmd ?? ["node", "--test", path.join(CHECKS_DIR, entry.selftest)]));
    const goTests = goEntries.length ? await runGo(goEntries) : [];
    for (const one of goTests) results.set(one.name, one);
  } else {
    for (const entry of entries.filter((e) => e.runner !== "go")) {
      if (entry.runner === "verifiers") results.set(entry.name, await runVerifiers(entry, options));
      else if (entry.runner === "command") results.set(entry.name, runCommand(entry, entry.cmd));
      else if (entry.runner === "eslint" || entry.runner === "stylelint") results.set(entry.name, runCommand(entry, ["node", "--test", path.join(CHECKS_DIR, entry.selftest)]));
      else results.set(entry.name, { name: entry.name, status: "fail", ms: 0, output: `unknown runner ${entry.runner}` });
    }
    for (const one of await goDone) results.set(one.name, one);
  }
  return entries.map((e) => results.get(e.name));
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const registry = readRegistry();
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
  console.log(`Running ${entries.length} of ${selection.length} checks${options.selftest ? " (planted-violation tests)" : ""}\n`);
  return printReport(await runChecks(entries, options));
}

if (process.argv[1] && path.resolve(process.argv[1]) === import.meta.filename) process.exit(await main());
