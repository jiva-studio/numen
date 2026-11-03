import { runTool } from "../../shared/tools.mjs";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { readPluginManifests } from "../../shared/manifests.mjs";
import { writePluginIndex } from "./core.mjs";
import {
  MODULES_DIR,
  REPO_ROOT,
  exec,
  failOnFindings,
  getGoReferences,
  getModuleSpecifiers,
  parsePackageJson,
  listShippedFiles,
  readSource,
  requireMatches,
} from "./helpers.mjs";

export function verifyPluginsStatic() {
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  readPluginManifests(pluginsDir);

  if (fs.existsSync(path.join(pluginsDir, "shared"))) {
    throw new Error("plugins/shared exists: a plugin shares code only through the SDK");
  }

  const coreLeak = runTool("git", ["grep", "-rn", "github.com/jiva-studio/numen/libs/core", "plugins/"], { cwd: REPO_ROOT });
  const lines = (coreLeak.stdout ? coreLeak.stdout.toString() : "").split("\n").filter(Boolean);
  const nonCommentLeaks = lines.filter((l) => !l.includes(".golangci.yml") && !l.includes("README"));
  if (nonCommentLeaks.length > 0) {
    throw new Error("Plugin imports core package:\n" + nonCommentLeaks.join("\n"));
  }
}

/** The packages a plugin depends on; their `package.json` `exports` declare the entries a plugin may name. */
const PLUGIN_PACKAGES = ["@numen/plugin-sdk", "@numen/ui"];

/** A build configuration file, which may name the build entries a package declares. */
const BUILD_CONFIGURATION = /\.config\.[cm]?[jt]s$/;

/** The entry a plugin file may not name: `testing` is the test entry a package declares, a stylesheet is an asset. */
const TEST_ENTRY = "./testing";

/**
 * Lists the specifiers of `files` that reach into a plugin package beyond its main entry. `files`
 * holds `{ file, literals }` with each literal as `{ line, value }`, and `exportsByPackage` maps a
 * package name to the keys and targets of its `exports`. A deep path passes when the package
 * declares it and it is the test entry, a stylesheet, or named from a build configuration file.
 */
export function findPluginEntryViolations(files, exportsByPackage) {
  const found = [];
  for (const { file, literals } of files) {
    for (const { line, value } of literals) {
      const name = PLUGIN_PACKAGES.find((one) => value.startsWith(one + "/"));
      if (!name) continue;
      const entry = "." + value.slice(name.length);
      const target = exportsByPackage[name]?.[entry];
      if (target === undefined) {
        found.push({ file, line, symbol: `${JSON.stringify(value)} is not an entry ${name} declares` });
        continue;
      }
      const targetPath = typeof target === "string" ? target : JSON.stringify(target);
      const isAllowed = entry === TEST_ENTRY || /\.css\b/.test(targetPath) || BUILD_CONFIGURATION.test(path.basename(file));
      if (!isAllowed) found.push({ file, line, symbol: `${JSON.stringify(value)} reaches past the main entry of ${name}` });
    }
  }
  return found;
}

/** The `exports` map of each plugin package, read from the `package.json` under `libs/` that names it. */
export function readPluginPackageExports() {
  const result = {};
  for (const file of listShippedFiles(["libs"]).filter((one) => path.basename(one) === "package.json" && !one.includes("node_modules/"))) {
    const manifest = parsePackageJson(fs.readFileSync(path.join(REPO_ROOT, file), "utf8"));
    if (PLUGIN_PACKAGES.includes(manifest.name)) result[manifest.name] = manifest.exports ?? {};
  }
  return result;
}

/** A plugin imports `@numen/plugin-sdk` and `@numen/ui` through their main entry, and from the test entry, stylesheets and build entries a package declares. */
export function verifyPluginsUseMainEntry() {
  const exportsByPackage = readPluginPackageExports();
  for (const name of PLUGIN_PACKAGES) requireMatches(`exports of ${name}`, Object.keys(exportsByPackage[name] ?? {}).length, 1);
  const files = [];
  let named = 0;
  for (const file of listShippedFiles(["plugins"])) {
    if (!/\.(ts|tsx|mts|cts|js|mjs|cjs|vue)$/.test(file)) continue;
    const source = readSource(file);
    if (!source) continue;
    const literals = source.literals.filter(({ value }) => PLUGIN_PACKAGES.some((one) => value === one || value.startsWith(one + "/")));
    named += literals.length;
    files.push({ file, literals });
  }
  requireMatches("plugin references to the plugin packages", named, 100);
  failOnFindings("A plugin reaches past the main entry of @numen/plugin-sdk or @numen/ui", findPluginEntryViolations(files, exportsByPackage));
}

/** Plugin backend internal layers (inner packages under backend/<pkg> must not import backend/service or backend/engine). */
function findPluginBackendLayerInversions() {
  const found = [];
  for (const file of listShippedFiles(["plugins/*.go"])) {
    const m = file.match(/^plugins\/([^/]+)\/backend\/([^/]+)\//);
    if (!m) continue;
    const [, plugin, subpkg] = m;
    if (["cmd", "service", "engine", "testsupport"].includes(subpkg)) continue;
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    const { imports } = getGoReferences(text);
    for (const { line, value } of imports) {
      if (
        value.endsWith("/backend/service") ||
        value.includes("/backend/service/") ||
        value.endsWith("/backend/engine") ||
        value.includes("/backend/engine/")
      ) {
        found.push({ file, line, symbol: `inner package backend/${subpkg} imports outer ${value} in plugins/${plugin}` });
      }
    }
  }
  return found;
}

export function verifyPluginBackendInternalLayers() {
  failOnFindings("Plugin backend inner package imports outer service/engine package", findPluginBackendLayerInversions());
}

/**
 * Golden Template & Config Drift Verification for Standalone Plugins.
 */
function parseEnabledGoLinters(yamlText) {
  const m = yamlText.match(/enable:\s*\n((?:\s*-\s*[\w-]+\s*\n)+)/);
  if (!m) return [];
  return m[1]
    .split("\n")
    .map((l) => l.replace(/^\s*-\s*/, "").trim())
    .filter(Boolean)
    .sort();
}

function findPluginConfigDrift() {
  const found = [];
  const exampleDir = path.join(REPO_ROOT, "plugins/.example");
  const rootGolangci = path.join(REPO_ROOT, ".golangci.yml");
  const exampleGolangci = path.join(exampleDir, ".golangci.yml");
  const baseEslint = path.join(REPO_ROOT, "tools/lint-layers/base-eslint.mjs");
  const exampleEslint = path.join(exampleDir, "eslint.config.js");

  if (fs.existsSync(rootGolangci) && fs.existsSync(exampleGolangci)) {
    const rootLinters = parseEnabledGoLinters(fs.readFileSync(rootGolangci, "utf8"));
    const exLinters = parseEnabledGoLinters(fs.readFileSync(exampleGolangci, "utf8"));
    for (const l of rootLinters) {
      if (!exLinters.includes(l)) {
        found.push({ file: "plugins/.example/.golangci.yml", line: 1, symbol: `missing linter '${l}' enabled in root .golangci.yml` });
      }
    }
    for (const l of exLinters) {
      if (!rootLinters.includes(l)) {
        found.push({ file: "plugins/.example/.golangci.yml", line: 1, symbol: `extra linter '${l}' not in root .golangci.yml` });
      }
    }
  }

  if (fs.existsSync(baseEslint) && fs.existsSync(exampleEslint)) {
    const exText = fs.readFileSync(exampleEslint, "utf8");
    const requiredSelectors = ["Date", "Math", "getBoundingClientRect", "matchMedia", "lib/", "model/", "api/wire"];
    for (const token of requiredSelectors) {
      if (!exText.includes(token)) {
        found.push({ file: "plugins/.example/eslint.config.js", line: 1, symbol: `missing rule selector '${token}' from base-eslint.mjs` });
      }
    }
    if (getModuleSpecifiers(exampleEslint, exText).some((s) => s.startsWith(".")) || exText.includes("@numen/plugin-sdk/eslint")) {
      found.push({ file: "plugins/.example/eslint.config.js", line: 1, symbol: "plugins/.example/eslint.config.js must be 100% self-contained without ../ paths or SDK coupling" });
    }
  }

  const goConfigCopies = ["plugins/.example", "libs/plugin-sdk-go"];
  const pluginsRoot = path.join(REPO_ROOT, "plugins");
  for (const entry of fs.readdirSync(pluginsRoot, { withFileTypes: true })) {
    if (entry.isDirectory() && !entry.name.startsWith(".") && fs.existsSync(path.join(pluginsRoot, entry.name, "go.mod"))) {
      goConfigCopies.push(`plugins/${entry.name}`);
    }
  }
  for (const name of [".golangci.yml", ".golangci-structure.yml"]) {
    const rootFile = path.join(REPO_ROOT, name);
    if (!fs.existsSync(rootFile)) {
      found.push({ file: name, line: 1, symbol: "the root Go lint configuration is missing" });
      continue;
    }
    const rootText = fs.readFileSync(rootFile, "utf8");
    for (const dir of goConfigCopies) {
      const copy = path.join(REPO_ROOT, dir, name);
      if (!fs.existsSync(copy) || fs.readFileSync(copy, "utf8") !== rootText) {
        found.push({ file: `${dir}/${name}`, line: 1, symbol: `must be a byte-identical copy of the root ${name}` });
      }
    }
  }

  for (const parentConfig of ["plugins/.golangci.yml", "plugins/eslint.config.js"]) {
    if (fs.existsSync(path.join(REPO_ROOT, parentConfig))) {
      found.push({ file: parentConfig, line: 1, symbol: "shared parent config outside plugin directory (each plugin must carry its own copy of plugins/.example/)" });
    }
  }

  const pluginsDir = path.join(REPO_ROOT, "plugins");
  const pluginNames = fs
    .readdirSync(pluginsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith(".") && e.name !== "node_modules")
    .map((e) => e.name)
    .sort();

  const checkGoldenFile = (plugin, fileName) => {
    const goldenPath = path.join(exampleDir, fileName);
    const targetPath = path.join(pluginsDir, plugin, fileName);
    const relTarget = `plugins/${plugin}/${fileName}`;
    if (!fs.existsSync(goldenPath)) return;
    if (!fs.existsSync(targetPath)) {
      found.push({ file: relTarget, line: 1, symbol: `missing standalone config (must match plugins/.example/${fileName})` });
      return;
    }
    const goldenText = fs.readFileSync(goldenPath, "utf8");
    const targetText = fs.readFileSync(targetPath, "utf8");
    if (goldenText !== targetText) {
      found.push({ file: relTarget, line: 1, symbol: `config drifted from golden template plugins/.example/${fileName}` });
    }
  };

  for (const plugin of pluginNames) {
    const pDir = path.join(pluginsDir, plugin);
    checkGoldenFile(plugin, "Makefile");
    if (fs.existsSync(path.join(pDir, "go.mod"))) {
      checkGoldenFile(plugin, ".golangci.yml");
    }
    if (fs.existsSync(path.join(pDir, "package.json"))) {
      checkGoldenFile(plugin, "eslint.config.js");
      checkGoldenFile(plugin, "tsconfig.json");
    }
  }

  return found;
}

export function verifyPluginConfigGoldenParity() {
  failOnFindings("Plugin configuration missing or drifted from golden template plugins/.example/", findPluginConfigDrift());
}

/** Runs ESLint across each frontend plugin directory using its own `eslint.config.js`. */
export function verifyPluginsFrontend() {
  const pluginsDir = path.join(REPO_ROOT, "plugins");
  const exampleConfig = path.join(pluginsDir, ".example", "eslint.config.js");
  if (!fs.existsSync(exampleConfig)) {
    throw new Error("Missing golden frontend config: plugins/.example/eslint.config.js");
  }
  const eslintBin = path.join(REPO_ROOT, "node_modules/.bin/eslint");
  const frontendPlugins = fs
    .readdirSync(pluginsDir, { withFileTypes: true })
    .filter(
      (e) =>
        e.isDirectory() &&
        !e.name.startsWith(".") &&
        e.name !== "node_modules" &&
        fs.existsSync(path.join(pluginsDir, e.name, "frontend")),
    )
    .map((e) => e.name)
    .sort();

  const failures = [];
  for (const plugin of frontendPlugins) {
    const pluginDir = path.join(pluginsDir, plugin);
    const configPath = path.join(pluginDir, "eslint.config.js");
    if (!fs.existsSync(configPath)) {
      failures.push(`plugins/${plugin}: missing eslint.config.js`);
      continue;
    }
    const res = spawnSync(eslintBin, ["."], { cwd: pluginDir, encoding: "utf8" });
    if (res.status !== 0) {
      failures.push(`plugins/${plugin}:\n${((res.stdout || "") + (res.stderr || "")).trim()}`);
    }
  }
  if (failures.length > 0) {
    throw new Error(failures.join("\n\n"));
  }
}

export const PLUGIN_VERIFIERS = [
  {
    id: "plugin-config-golden-parity",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml",
    run: verifyPluginConfigGoldenParity,
  },
  {
    id: "plugin-backend-layers",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Backend Internal Hexagonal Layer Direction",
    run: verifyPluginBackendInternalLayers,
  },
  {
    id: "plugins-independence",
    isGoTest: true,
    scope: "plugins",
    module: "plugins",
    name: "Plugin Independence Architecture Suite (14 platform targets)",
    coveredBy: "go-structure",
    run: () => {
      writePluginIndex();
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
    id: "plugins-use-main-entry",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Import the SDK and UI Through Their Main Entry (only the test entry, stylesheets and build entries a package declares in exports)",
    run: verifyPluginsUseMainEntry,
  },
  {
    id: "plugins-frontend",
    isLint: true,
    scope: "plugins",
    module: "plugins",
    name: "Plugin Frontend Contract & Boundaries (no @/**, no @plugins/**)",
    run: verifyPluginsFrontend,
  },
];
