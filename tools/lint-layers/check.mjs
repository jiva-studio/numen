#!/usr/bin/env node
/**
 * Numen Architectural Layer & Boundary Validator
 *
 * Validates dependency flow, layer isolation, and architectural invariants
 * across libraries (libs), native/frontend plugins (plugins), and applications (apps).
 *
 * Usage:
 *   node check.mjs [target...]
 *   node check.mjs --list
 *   node check.mjs --help
 *
 * Scopes:
 *   all       - Run all architectural layer checks (default)
 *   libs      - All library boundaries (core hexagon, notes, ui, shell and plugin host isolation, sdk isolation)
 *   plugins   - Native plugin sandboxing, non-leakage, and frontend contracts
 *   apps      - Application FSD layer hierarchies (desktop editor, flashcards)
 *
 * Modules:
 *   core, notes, protocol, kinds, ui, shell, host, wire, sdk, plugins, boundary, desktop, editor, flashcards
 */
import { spawnSync } from "node:child_process";
import process from "node:process";
import { CORE_VERIFIERS } from "./checks/core.mjs";
import { FRONTEND_VERIFIERS } from "./checks/frontend.mjs";
import { REPO_ROOT } from "./checks/helpers.mjs";
import { PLUGIN_VERIFIERS } from "./checks/plugins.mjs";

const VERIFIERS = [
  ...CORE_VERIFIERS,
  ...FRONTEND_VERIFIERS.filter((v) => v.scope === "libs"),
  ...PLUGIN_VERIFIERS,
  ...FRONTEND_VERIFIERS.filter((v) => v.scope === "apps"),
];

const SCOPES = ["all", "libs", "plugins", "apps"];
const MODULE_NAMES = Array.from(new Set(VERIFIERS.map((v) => v.module)));

/** The options of a run; every other argument is a target. */
const FLAGS = ["--list", "--help", "-h"];
const flags = new Set(process.argv.slice(2).filter((arg) => FLAGS.includes(arg)));
const args = process.argv.slice(2).filter((arg) => !FLAGS.includes(arg));

if (flags.has("--help") || flags.has("-h")) {
  console.log("Usage: node check.mjs [target...] [--list] [--help]");
  console.log("");
  console.log("Targets can be scopes or module names:");
  console.log("  Scopes:  " + SCOPES.join(", "));
  console.log("  Modules: " + MODULE_NAMES.join(", "));
  console.log("");
  console.log("Examples:");
  console.log("  node check.mjs               # validate all layers across monorepo");
  console.log("  node check.mjs libs          # validate library layers only");
  console.log("  node check.mjs plugins       # validate plugin isolation and boundaries");
  console.log("  node check.mjs apps          # validate application FSD layers");
  console.log("  node check.mjs core ui       # validate core and UI library");
  process.exit(0);
}

if (flags.has("--list")) {
  console.log("Registered architectural verifiers:");
  for (const v of VERIFIERS) {
    console.log("  [" + v.scope.padEnd(7) + " / " + v.module.padEnd(8) + "] " + v.id + ": " + v.name);
  }
  process.exit(0);
}

function getChangedModules() {
  const currentBranch = spawnSync("git", ["rev-parse", "--abbrev-ref", "HEAD"], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  }).stdout?.trim();

  const uncommitted =
    spawnSync("git", ["diff", "--name-only", "HEAD"], {
      cwd: REPO_ROOT,
      encoding: "utf8",
    }).stdout?.split("\n").map((f) => f.trim()).filter(Boolean) || [];

  let base = process.env.BASE_REF;
  if (!base) {
    base = currentBranch === "main" ? "HEAD~1" : "origin/main";
  }

  let res = spawnSync("git", ["diff", "--name-only", `${base}...HEAD`], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  });
  if (res.status !== 0) {
    res = spawnSync("git", ["diff", "--name-only", "HEAD~1...HEAD"], {
      cwd: REPO_ROOT,
      encoding: "utf8",
    });
  }

  const committed = (res.stdout || "").split("\n").map((f) => f.trim()).filter(Boolean);
  const files = Array.from(new Set([...uncommitted, ...committed]));

  const matched = new Set();
  for (const file of files) {
    if (file.startsWith("libs/core/")) matched.add("core");
    if (file.startsWith("libs/notes/")) matched.add("notes");
    if (file.startsWith("libs/protocol/") || file.startsWith("tools/lint-layers/")) matched.add("protocol");
    if (/^(libs\/(core|notes|plugin-sdk-go|plugin-sdk-ts|ui)|apps\/desktop\/editor|tools\/lint-layers)\//.test(file)) matched.add("kinds");
    if (file.startsWith("tools/lint-layers/")) ["ui", "plugins"].forEach((one) => matched.add(one));
    if (file.startsWith("libs/plugin-sdk-go/") || file.startsWith("libs/plugin-sdk-ts/")) matched.add("sdk");
    if (file.startsWith("libs/ui/core/")) matched.add("ui");
    if (file.startsWith("libs/ui/shell/")) matched.add("shell");
    if (file.startsWith("libs/plugin-host-ts/")) matched.add("host");
    if (file.startsWith("plugins/")) matched.add("plugins");
    if (file.startsWith("apps/desktop/")) matched.add("desktop");
    if (file.startsWith("apps/")) matched.add("apps");
    if (file.startsWith("apps/") || file.startsWith("plugins/")) matched.add("boundary");
  }
  return Array.from(matched);
}

function resolveSelectedVerifiers(targets) {
  if (targets.includes("--changed-only")) {
    const changedModules = getChangedModules();
    if (changedModules.length === 0) {
      return [];
    }
    return VERIFIERS.filter((v) => changedModules.includes(v.module));
  }
  if (targets.length === 0 || targets.includes("all")) {
    return VERIFIERS;
  }
  const selected = new Set();
  for (const target of targets) {
    if (SCOPES.includes(target)) {
      VERIFIERS.filter((v) => v.scope === target).forEach((v) => selected.add(v));
    } else if (MODULE_NAMES.includes(target)) {
      VERIFIERS.filter((v) => v.module === target).forEach((v) => selected.add(v));
    } else {
      const byId = VERIFIERS.find((v) => v.id === target);
      if (byId) {
        selected.add(byId);
      } else {
        console.error("Unknown target: " + target);
        console.error("Run with --help to see available targets and scopes.");
        process.exit(2);
      }
    }
  }
  return VERIFIERS.filter((v) => selected.has(v));
}

const activeVerifiers = resolveSelectedVerifiers(args);

if (activeVerifiers.length === 0) {
  console.log("\x1b[1mNumen Architectural Layer & Boundary Validator\x1b[0m");
  console.log("No architectural checks required for changed files.\n");
  process.exit(0);
}

console.log("\x1b[1mNumen Architectural Layer & Boundary Validator\x1b[0m");
console.log("Selected targets: " + (args.length === 0 ? "all" : args.join(", ")));
console.log("Verifiers count:  " + activeVerifiers.length + "\n");

let exitCode = 0;
const results = [];
let currentScope = "";

for (const verifier of activeVerifiers) {
  if (verifier.scope !== currentScope) {
    currentScope = verifier.scope;
    console.log("\n\x1b[1m\x1b[36m=== Scope: " + currentScope.toUpperCase() + " ===\x1b[0m");
  }

  const start = Date.now();
  process.stdout.write("  • " + verifier.name + "... ");
  try {
    verifier.run();
    const duration = ((Date.now() - start) / 1000).toFixed(1) + "s";
    console.log("\x1b[32m[PASS]\x1b[0m (" + duration + ")");
    results.push({ name: verifier.name, status: "PASS", duration });
  } catch (err) {
    const duration = ((Date.now() - start) / 1000).toFixed(1) + "s";
    console.log("\x1b[31m[FAIL]\x1b[0m (" + duration + ")");
    if (err.stdout) console.error(err.stdout.toString());
    if (err.stderr) console.error(err.stderr.toString());
    if (!err.stdout && !err.stderr && err.message) console.error("    " + err.message);
    results.push({ name: verifier.name, status: "FAIL", duration, error: err.message });
    exitCode = 1;
  }
}

console.log("\n\x1b[1m=== Validation Summary ===\x1b[0m");
for (const res of results) {
  const badge = res.status === "PASS" ? "\x1b[32m[PASS]\x1b[0m" : "\x1b[31m[FAIL]\x1b[0m";
  console.log("  " + badge + " " + res.name + " (" + res.duration + ")");
}

if (exitCode === 0) {
  console.log("\n\x1b[32m\x1b[1mAll architectural layer boundaries and invariants are clean and intact!\x1b[0m\n");
} else {
  console.log("\n\x1b[31m\x1b[1mArchitecture validation failed with layer or boundary violations!\x1b[0m\n");
}

process.exit(exitCode);
