#!/usr/bin/env node
import { execSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

const SCRIPT_DIR = path.dirname(new URL(import.meta.url).pathname);
const MODULES_DIR = path.resolve(SCRIPT_DIR, "../..");
const REPO_ROOT = path.resolve(MODULES_DIR, "..");

const args = process.argv.slice(2);
const runAll = args.length === 0 || args.includes("--all");
const checkBackend = runAll || args.includes("--backend");
const checkPlugins = runAll || args.includes("--plugins");
const checkFrontend = runAll || args.includes("--frontend");
const fastMode = args.includes("--fast");

let exitCode = 0;
const results = [];

function logSection(title) {
  console.log("\n\x1b[1m\x1b[36m=== " + title + " ===\x1b[0m");
}

function runStep(name, fn) {
  const start = Date.now();
  process.stdout.write("  • " + name + "... ");
  try {
    fn();
    const duration = ((Date.now() - start) / 1000).toFixed(1) + "s";
    console.log("\x1b[32m✟ PASS\x1b[0m (" + duration + ")");
    results.push({ name, status: "PASS", duration });
  } catch (err) {
    const duration = ((Date.now() - start) / 1000).toFixed(1) + "s";
    console.log("\x1b[31m✜ FAIL\x1b[0m (" + duration + ")");
    if (err.stdout) console.error(err.stdout.toString());
    if (err.stderr) console.error(err.stderr.toString());
    if (!err.stdout && !err.stderr && err.message) console.error("    " + err.message);
    results.push({ name, status: "FAIL", duration, error: err.message });
    exitCode = 1;
  }
}

function exec(cmd, cwd = REPO_ROOT) {
  return execSync(cmd, { cwd, stdio: ["ignore", "pipe", "pipe"] });
}

console.log("\x1b[1mNumen Architectural Layer & Boundary Validator\x1b[0m");
console.log("Root: " + REPO_ROOT + "\n");

// --------------------------------- Backend ---------------------------------
if (checkBackend) {
  logSection("1. Backend Hexagonal Layer Architecture (Go)");

  runStep("Core Hexagonal Layers & Inward Dependencies (layers_test.go)", () => {
    exec('go test -v ./container -run "TestTheLayersAreWhatTheyAre|TestNoPurePackageIsTestedThroughAnAdapter|TestNoAdapterNamesThePortItSatisfies|TestATestOfTheCoreBuildingAnAdapterStandsOutsideIt|TestNothingOfTheCoreReachesTheMachine|TestNoInnerLayerImportsATransport|TestNothingOfTheCoreLogs|TestNothingOfTheCoreReadsTMeMachinesClock|TestTheCoresIdentitiesAreItsOwn|TestEveryAdapterServingTheSchemaIsDriving|TestEveryPortIsAskedForSomewhereElse|TestEveryTypePortDeclaresIsNamedSomewhereElse|TestEveryPortMethodIsCalledSomewhereElse|TestNoPackageIsImportedUnderItsLayer|TestTheCoresPublicAdaptersAreTheseAndNoOthers|TestTheCoresHeldAdaptersAreTheseAndNoOthers|TestEveryInstantTheCoreCarriesIsATime|TestTheIdentitiesAPortAsksForAreTheCoresOwn|TestContainerDoesNotDowncastPluginHost"', path.join(MODULES_DIR, "libs/core"));
  });

  runStep("Go Depguard Boundaries (Core & Notes & SDK)", () => {
    exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/core"));
    exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/notes"));
    exec("golangci-lint run --issues-exit-code 1 ./...", path.join(MODULES_DIR, "libs/plugin-sdk-go"));
  });
}

// -------------------------------- Plugins --------------------------------
if (checkPlugins) {
  logSection("2. Native Plugin Isolation & Sandboxing");

  runStep("Plugin Independence Architecture Suite (14 platform targets)", () => {
    exec('go test -v ./container -run "Plugin"', path.join(MODULES_DIR, "libs/core"));
  });

  runStep("Plugin Non-Leakage Static Verification", () => {
    const pluginsDir = path.join(MODULES_DIR, "plugins");
    const entries = fs.readdirSync(pluginsDir, { withFileTypes: true });
    const pluginNames = entries.filter((e) => e.isDirectory() && !e.name.startsWith(".")).map((e) => e.name);

    for (const plugin of pluginNames) {
      const pDir = path.join(pluginsDir, plugin);
      const manifestPath = path.join(pDir, "manifest.json");
      if (fs.existsSync(manifestPath)) {
        const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
        if (!manifest.id || !manifest.name || !manifest.version) {
          throw new Error("Plugin " + plugin + " has invalid manifest.json");
        }
      }
    }

    // Check no plugin imports core
    const coreLeak = spawnSync("git", ["grep", "-rn", "github.com/jiva-studio/numen/modules/libs/core", "modules/plugins/"], { cwd: REPO_ROOT });
    const lines = (coreLeak.stdout ? coreLeak.stdout.toString() : "").split("\n").filter(Boolean);
    const nonCommentLeaks = lines.filter((l) => !l.includes(".golangci.yml") && !l.includes("README"));
    if (nonCommentLeaks.length > 0) {
      throw new Error("Plugin imports core package: " + nonCommentLeaks.join(", "));
    }

    // Check no cross-plugin imports
    for (const plugin of pluginNames) {
      for (const other of pluginNames) {
        if (plugin === other) continue;
        const crossLeak = spawnSync("git", ["grep", "-rn", "github.com/jiva-studio/numen/modules/plugins/" + other, "modules/plugins/" + plugin], { cwd: REPO_ROOT });
        const leaks = (crossLeak.stdout ? crossLeak.stdout.toString() : "").trim();
        if (leaks) {
          throw new Error("Cross-plugin import detected: " + plugin + " imports " + other + "\n" + leaks);
        }
      }
    }
  });

  runStep("Plugin Frontend Boundaries (@numen/* contract, no @/**, no @plugins/**)", () => {
    exec("../node_modules/.bin/eslint .", path.join(MODULES_DIR, "plugins"));
  });
}

// -------------------------------- Frontend -------------------------------
if (checkFrontend) {
  logSection("3. Frontend FSD Layers & Library Boundaries");

  runStep("UI Library Domain Isolation (0 domain, 0 protocol, 0 wire, 0 apps)", () => {
    const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire)|modules/apps)", "modules/libs/ui/src/"], { cwd: REPO_ROOT });
    const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
    if (leaks) {
      throw new Error("UI library imports domain, wire, protocol or apps:\n" + leaks);
    }
    exec("npm run lint", path.join(MODULES_DIR, "libs/ui"));
  });

  if (!fastMode) {
    runStep("Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> widgets -> pages -> app)", () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/editor"));
    });

    runStep("Desktop Flashcards FSD Layer Hierarchy", () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/flashcards"));
    });
  } else {
    console.log("  ⁹ Fast mode: skipped full type-aware desktop linter (use without --fast for full coverage)");
  }
}

// ------------------------------ Summary ------------------------------
console.log("\n\x1b[1m=== Validation Summary ===\x1b[0m");
for (const res of results) {
  const icon = res.status === "PASS" ? "\x1b[32m✼\x1b[0m" : "\x1b[31m✜�x1b[0m";
  console.log("  " + icon + " " + res.name + " (" + res.duration + ")");
}

if (exitCode === 0) {
  console.log("\n\x1b[32m\x1b[1mAll architectural layer boundaries and invariants are clean and intact!\x1b[0m\n");
} else {
  console.log("\n\x1b[31m\x1b[1mArchitecture validation failed with layer or boundary violations!\x1b[0m\n");
}

process.exit(exitCode);
