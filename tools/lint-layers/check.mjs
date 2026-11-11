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
 *   apps      - Application FSD layer hierarchies (desktop editor, flashcards, mobile)
 *
 * Modules:
 *   the `module` of each verifier; `node check.mjs --help` lists them
 */
import process from "node:process";
import { APP_LOGIC_VERIFIERS } from "./checks/appLogic.mjs";
import { CORE_CLIENT_VERIFIERS } from "./checks/coreClient.mjs";
import { DEPENDENCY_VERIFIERS } from "./checks/dependencies.mjs";
import { DISCIPLINE_VERIFIERS } from "./checks/discipline.mjs";
import { DOCUMENT_VERIFIERS } from "./checks/documents.mjs";
import { FRONTEND_VERIFIERS } from "./checks/frontend.mjs";
import { LIBRARY_VERIFIERS } from "./checks/libraries.mjs";
import { LINT_CONFIG_VERIFIERS } from "./checks/lintConfig.mjs";
import { PLUGIN_VERIFIERS } from "./checks/plugins.mjs";
import { REGISTRATION_VERIFIERS } from "./checks/registrations.mjs";
import { SDK_VERIFIERS } from "./checks/sdkProtocol.mjs";
import { STYLE_VERIFIERS } from "./checks/styles.mjs";
import { STORY_VERIFIERS } from "./checks/stories.mjs";
import { STORY_LINT_VERIFIERS } from "./checks/storyLint.mjs";
import { assertGolangciLintBuild } from "./checks/tools.mjs";
import { TYPECHECK_SCOPE_VERIFIERS } from "./checks/typecheckScope.mjs";
import { FRONTEND_GRAPH_VERIFIERS } from "./checks/frontendGraph.mjs";
import { FRONTEND_LAYOUT_VERIFIERS } from "./checks/frontendLayout.mjs";

const VERIFIERS = [
  ...SDK_VERIFIERS,
  ...LIBRARY_VERIFIERS,
  ...CORE_CLIENT_VERIFIERS,
  ...DEPENDENCY_VERIFIERS,
  ...FRONTEND_GRAPH_VERIFIERS.filter((v) => v.scope === "libs"),
  ...FRONTEND_LAYOUT_VERIFIERS.filter((v) => v.scope === "libs"),
  ...STYLE_VERIFIERS,
  ...APP_LOGIC_VERIFIERS.filter((v) => v.scope === "libs"),
  ...DOCUMENT_VERIFIERS,
  ...REGISTRATION_VERIFIERS,
  ...STORY_VERIFIERS,
  ...STORY_LINT_VERIFIERS,
  ...FRONTEND_VERIFIERS.filter((v) => v.scope === "libs"),
  ...LINT_CONFIG_VERIFIERS,
  ...TYPECHECK_SCOPE_VERIFIERS,
  ...DISCIPLINE_VERIFIERS.filter((v) => v.scope === "libs"),
  ...PLUGIN_VERIFIERS,
  ...DISCIPLINE_VERIFIERS.filter((v) => v.scope === "plugins"),
  ...FRONTEND_LAYOUT_VERIFIERS.filter((v) => v.scope === "plugins"),
  ...FRONTEND_VERIFIERS.filter((v) => v.scope === "apps"),
  ...FRONTEND_GRAPH_VERIFIERS.filter((v) => v.scope === "apps"),
  ...APP_LOGIC_VERIFIERS.filter((v) => v.scope === "apps"),
  ...FRONTEND_LAYOUT_VERIFIERS.filter((v) => v.scope === "apps"),
];

const SCOPES = ["all", "libs", "plugins", "apps"];
const MODULE_NAMES = Array.from(new Set(VERIFIERS.map((v) => v.module)));

/** The options of a run; every other argument is a target. */
const FLAGS = ["--list", "--help", "-h", "--without-lint"];
const flags = new Set(process.argv.slice(2).filter((arg) => FLAGS.includes(arg)));
const args = process.argv.slice(2).filter((arg) => !FLAGS.includes(arg));

if (flags.has("--help") || flags.has("-h")) {
  console.log("Usage: node check.mjs [target...] [--without-lint] [--list] [--help]");
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

/** The verifiers a scope, a module, a verifier id or the full run selects. */
function resolveSelectedVerifiers(targets) {
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

const selectedVerifiers = resolveSelectedVerifiers(args);
const activeVerifiers = selectedVerifiers
  // `isLint` runs the ESLint or golangci-lint configuration of a package, which its own lint target runs as well.
  .filter((v) => !(flags.has("--without-lint") && v.isLint));

console.log("\x1b[1mNumen Architectural Layer & Boundary Validator\x1b[0m");
console.log("Selected targets: " + (args.length === 0 ? "all" : args.join(", ")));
console.log("Verifiers count:  " + activeVerifiers.length + "\n");

if (activeVerifiers.some((v) => v.tools?.includes("golangci-lint"))) {
  try {
    assertGolangciLintBuild();
  } catch (err) {
    console.error("\x1b[31mTool version check failed:\x1b[0m " + err.message);
    process.exit(1);
  }
}

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
    await verifier.run();
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
