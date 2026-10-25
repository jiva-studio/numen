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
 *   libs      - All library boundaries (core hexagon, notes, ui isolation, sdk isolation)
 *   plugins   - Native plugin sandboxing, non-leakage, and frontend contracts
 *   apps      - Application FSD layer hierarchies (desktop editor, flashcards)
 *
 * Modules:
 *   core, notes, ui, wire, sdk, plugins, desktop, editor, flashcards
 */
import { execSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const SCRIPT_DIR = path.dirname(new URL(import.meta.url).pathname);
const REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");
const MODULES_DIR = REPO_ROOT;

function exec(cmd, cwd = REPO_ROOT) {
  return execSync(cmd, { cwd, stdio: ["ignore", "pipe", "pipe"] });
}

function verifyPluginsStatic() {
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  const entries = fs.readdirSync(pluginsDir, { withFileTypes: true });
  const pluginNames = entries
    .filter((e) => e.isDirectory() && !e.name.startsWith("."))
    .map((e) => e.name);

  // 1. Verify manifest.json in each plugin
  for (const plugin of pluginNames) {
    const pDir = path.join(pluginsDir, plugin);
    const manifestPath = path.join(pDir, "manifest.json");
    if (fs.existsSync(manifestPath)) {
      const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
      if (!manifest.id || !manifest.name || !manifest.version) {
        throw new Error("Plugin " + plugin + " has invalid manifest.json (missing id/name/version)");
      }
    }
  }

  // 2. Verify plugins share no internal library folder
  if (fs.existsSync(path.join(pluginsDir, "shared"))) {
    throw new Error("plugins/shared exists: a plugin shares code only through the SDK");
  }

  // 3. Verify no plugin imports core
  const coreLeak = spawnSync("git", ["grep", "-rn", "github.com/jiva-studio/numen/libs/core", "plugins/"], { cwd: REPO_ROOT });
  const lines = (coreLeak.stdout ? coreLeak.stdout.toString() : "").split("\n").filter(Boolean);
  const nonCommentLeaks = lines.filter((l) => !l.includes(".golangci.yml") && !l.includes("README"));
  if (nonCommentLeaks.length > 0) {
    throw new Error("Plugin imports core package:\n" + nonCommentLeaks.join("\n"));
  }

  // 4. Verify no cross-plugin imports
  for (const plugin of pluginNames) {
    for (const other of pluginNames) {
      if (plugin === other) continue;
      const crossLeak = spawnSync("git", ["grep", "-rn", "github.com/jiva-studio/numen/plugins/" + other, "plugins/" + plugin], { cwd: REPO_ROOT });
      const leaks = (crossLeak.stdout ? crossLeak.stdout.toString() : "").trim();
      if (leaks) {
        throw new Error("Cross-plugin import detected: " + plugin + " imports " + other + "\n" + leaks);
      }
    }
  }
}

/** The files that state the boundary between applications and plugins, and so name both sides. */
const BOUNDARY_RULES = new Set([
  "apps/desktop/editor/eslint.config.js",
  "apps/desktop/internal/layers/linked_test.go",
  "plugins/eslint.config.js",
]);

/**
 * The directories a relative path in `file` is read from: the file's own directory, and every
 * package root above it, which is where a script or a test runs.
 */
function getPathBases(file) {
  const bases = [path.join(REPO_ROOT, path.dirname(file))];
  for (let dir = path.dirname(file); dir !== "."; dir = path.dirname(dir)) {
    const abs = path.join(REPO_ROOT, dir);
    if (fs.existsSync(path.join(abs, "package.json")) || fs.existsSync(path.join(abs, "go.mod"))) {
      bases.push(abs);
    }
  }
  return bases;
}

/**
 * Every tracked line under `from` that reaches the top-level tree `to`: a relative path that
 * resolves into it, or a match of one of `patterns`. The files in BOUNDARY_RULES are not read.
 */
function findCrossings(from, to, patterns) {
  const relative = `(\\.\\./)+${to}([/'"\`*]|$)`;
  const grep = spawnSync(
    "git",
    ["grep", "-nE", "-e", relative, ...patterns.flatMap((p) => ["-e", p]), "--", from],
    { cwd: REPO_ROOT, encoding: "utf8" },
  );
  const target = path.join(REPO_ROOT, to);
  const named = patterns.map((p) => new RegExp(p));
  return (grep.stdout || "").split("\n").filter(Boolean).filter((line) => {
    const [file, , ...rest] = line.split(":");
    if (BOUNDARY_RULES.has(file)) return false;
    const text = rest.join(":");
    if (named.some((p) => p.test(text))) return true;
    const bases = getPathBases(file);
    for (const match of text.matchAll(new RegExp(`((?:\\.\\./)+)${to}\\b`, "g"))) {
      if (bases.some((base) => path.resolve(base, match[1] + to) === target)) return true;
    }
    return false;
  });
}

/** An application knows no plugin: it compiles, imports, aliases and reads nothing under `plugins/`. */
function verifyAppsReachNoPlugin() {
  const crossings = findCrossings("apps", "plugins", [
    "@plugins\\b",
    "github\\.com/jiva-studio/numen/plugins\\b",
  ]);
  if (crossings.length > 0) {
    throw new Error("An application reaches into plugins/:\n" + crossings.join("\n"));
  }
}

/** A plugin knows no application: it compiles, imports, aliases and reads nothing under `apps/`. */
function verifyPluginsReachNoApp() {
  const crossings = findCrossings("plugins", "apps", [
    "github\\.com/jiva-studio/numen/apps\\b",
    "['\"]apps/",
    "['\"]@/",
    "['\"]@['\"]",
  ]);
  if (crossings.length > 0) {
    throw new Error("A plugin reaches into apps/:\n" + crossings.join("\n"));
  }
}

/** The packages of the core that belong to a plugin, as directories. */
const CORE_MOVED_DIRS = ["usecase/document"];

/** Every file under `dir` in the working tree, relative to the repository root. */
function listFiles(dir) {
  const abs = path.join(REPO_ROOT, dir);
  if (!fs.existsSync(abs)) return [];
  return fs
    .readdirSync(abs, { recursive: true, withFileTypes: true })
    .filter((e) => e.isFile())
    .map((e) => path.relative(REPO_ROOT, path.join(e.parentPath, e.name)));
}

/** The core holds none of the packages a domain plugin owns. */
function verifyCoreHoldsNoPluginPackage() {
  const found = [];
  for (const dir of CORE_MOVED_DIRS) {
    const files = listFiles(path.join("libs/core", dir));
    if (files.length > 0) found.push(`libs/core/${dir}/ (${files.length} files)`);
  }
  if (found.length > 0) {
    throw new Error("The core holds packages a plugin owns:\n" + found.join("\n"));
  }
}

function verifyUiDomainIsolation() {
  const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire)|apps)", "libs/ui/src/"], { cwd: REPO_ROOT });
  const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
  if (leaks) {
    throw new Error("UI library imports domain, wire, protocol or apps:\n" + leaks);
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/ui"));
}

const VERIFIERS = [
  {
    id: "core-hexagon",
    scope: "libs",
    module: "core",
    name: "Core Hexagonal Layers & Inward Dependencies (layers_test.go)",
    run: () => {
      exec(
        "go test -v ./container -run \"TestTheLayersAreWhatTheyAre|TestNoPurePackageIsTestedThroughAnAdapter|TestNoAdapterNamesThePortItSatisfies|TestATestOfTheCoreBuildingAnAdapterStandsOutsideIt|TestNothingOfTheCoreReachesTheMachine|TestNoInnerLayerImportsATransport|TestNothingOfTheCoreLogs|TestNothingOfTheCoreReadsTheMachinesClock|TestTheCoresIdentitiesAreItsOwn|TestEveryAdapterServingTheSchemaIsDriving|TestEveryPortIsAskedForSomewhereElse|TestEveryTypePortDeclaresIsNamedSomewhereElse|TestEveryPortMethodIsCalledSomewhereElse|TestNoPackageIsImportedUnderItsLayer|TestTheCoresPublicAdaptersAreTheseAndNoOthers|TestTheCoresHeldAdaptersAreTheseAndNoOthers|TestEveryInstantTheCoreCarriesIsATime|TestTheIdentitiesAPortAsksForAreTheCoresOwn|TestContainerDoesNotDowncastPluginHost\"",
        path.join(MODULES_DIR, "libs/core")
      );
    },
  },
  {
    id: "core-depguard",
    scope: "libs",
    module: "core",
    name: "Core Domain & UseCase Depguard Boundaries (.golangci.yml)",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/core"));
    },
  },
  {
    id: "notes-depguard",
    scope: "libs",
    module: "notes",
    name: "Notes Library Depguard Boundaries",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/notes"));
    },
  },
  {
    id: "core-holds-no-plugin-package",
    scope: "libs",
    module: "core",
    name: "Core Holds No Plugin Package (document)",
    run: verifyCoreHoldsNoPluginPackage,
  },
  {
    id: "sdk-isolation",
    scope: "libs",
    module: "sdk",
    name: "Plugin SDK Go Isolation (no core or app imports)",
    run: () => {
      exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/plugin-sdk-go"));
    },
  },
  {
    id: "ui-isolation",
    scope: "libs",
    module: "ui",
    name: "UI Library Domain & Wire Isolation (0 domain, 0 protocol, 0 wire, 0 apps)",
    run: verifyUiDomainIsolation,
  },
  {
    id: "plugins-independence",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Independence Architecture Suite (14 platform targets)",
    run: () => {
      exec("go test -v ./container -run \"Plugin\"", path.join(MODULES_DIR, "libs/core"));
    },
  },
  {
    id: "plugins-leakage",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Static Non-Leakage & Manifest Schema Validation",
    run: verifyPluginsStatic,
  },
  {
    id: "plugins-reach-no-app",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Reach No Application (no path, alias or import into apps/)",
    run: verifyPluginsReachNoApp,
  },
  {
    id: "plugins-frontend",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Frontend Contract & Boundaries (no @/**, no @plugins/**)",
    run: () => {
      exec("../node_modules/.bin/eslint .", path.join(MODULES_DIR, "plugins"));
    },
  },
  {
    id: "apps-reach-no-plugin",
    scope: "apps",
    module: "apps",
    name: "Applications Reach No Plugin (no path, alias, glob or import into plugins/)",
    run: verifyAppsReachNoPlugin,
  },
  {
    id: "desktop-editor-fsd",
    scope: "apps",
    module: "desktop",
    name: "Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> widgets -> pages -> app)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/editor"));
    },
  },
  {
    id: "desktop-flashcards-fsd",
    scope: "apps",
    module: "desktop",
    name: "Desktop Flashcards FSD Layer Hierarchy",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/flashcards"));
    },
  },
];

const SCOPES = ["all", "libs", "plugins", "apps"];
const MODULE_NAMES = Array.from(new Set(VERIFIERS.map((v) => v.module)));

const args = process.argv.slice(2);

if (args.includes("--help") || args.includes("-h")) {
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

if (args.includes("--list")) {
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
    if (file.startsWith("libs/plugin-sdk-go/")) matched.add("sdk");
    if (file.startsWith("libs/ui/")) matched.add("ui");
    if (file.startsWith("plugins/")) matched.add("plugins");
    if (file.startsWith("apps/desktop/")) matched.add("desktop");
    if (file.startsWith("apps/")) matched.add("apps");
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
  return Array.from(selected);
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
