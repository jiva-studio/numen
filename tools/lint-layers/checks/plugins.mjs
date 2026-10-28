import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import {
  MODULES_DIR,
  NOT_SHIPPED,
  REPO_ROOT,
  SCRIPT_EXTENSIONS,
  WINDOW_IDS,
  WINDOW_ROUTE_PREFIXES,
  exec,
  failOnFindings,
  findBoundaryReferences,
  findCrossings,
  findInCode,
  getGoReferences,
  getImportSpecifiers,
  getPackageNames,
  isWithin,
  listShippedFiles,
  readSource,
} from "./helpers.mjs";

/**
 * Every TypeScript or Vue import of a plugin that names another plugin: a relative path that
 * resolves into the other plugin's directory, or the other plugin's package name.
 */
function findCrossPluginImports(pluginNames) {
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  const packages = new Map();
  for (const plugin of pluginNames) {
    const manifest = path.join(pluginsDir, plugin, "package.json");
    if (fs.existsSync(manifest)) packages.set(plugin, JSON.parse(fs.readFileSync(manifest, "utf8")).name);
  }
  const found = [];
  for (const plugin of pluginNames) {
    const listed = spawnSync("git", ["ls-files", "--", `plugins/${plugin}/*.ts`, `plugins/${plugin}/*.vue`], {
      cwd: REPO_ROOT,
      encoding: "utf8",
    });
    for (const file of (listed.stdout || "").split("\n").filter(Boolean)) {
      const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
      for (const specifier of getImportSpecifiers(text)) {
        const other = specifier.startsWith(".")
          ? path.relative(pluginsDir, path.resolve(REPO_ROOT, path.dirname(file), specifier)).split(path.sep)[0]
          : [...packages].find(([, name]) => specifier === name || specifier.startsWith(name + "/"))?.[0];
        if (other && other !== plugin && pluginNames.includes(other)) found.push(`${file}: ${specifier}`);
      }
    }
  }
  return found;
}

export function verifyPluginsStatic() {
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  const entries = fs.readdirSync(pluginsDir, { withFileTypes: true });
  const pluginNames = entries
    .filter((e) => e.isDirectory() && !e.name.startsWith(".") && e.name !== "node_modules")
    .map((e) => e.name);

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

  if (fs.existsSync(path.join(pluginsDir, "shared"))) {
    throw new Error("plugins/shared exists: a plugin shares code only through the SDK");
  }

  const coreLeak = spawnSync("git", ["grep", "-rn", "github.com/jiva-studio/numen/libs/core", "plugins/"], { cwd: REPO_ROOT });
  const lines = (coreLeak.stdout ? coreLeak.stdout.toString() : "").split("\n").filter(Boolean);
  const nonCommentLeaks = lines.filter((l) => !l.includes(".golangci.yml") && !l.includes("README"));
  if (nonCommentLeaks.length > 0) {
    throw new Error("Plugin imports core package:\n" + nonCommentLeaks.join("\n"));
  }

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

  const frontendLeaks = findCrossPluginImports(pluginNames);
  if (frontendLeaks.length > 0) {
    throw new Error("Cross-plugin import detected:\n" + frontendLeaks.join("\n"));
  }
}

/** A plugin knows no application: it compiles, imports, aliases and reads nothing under `apps/`. */
export function verifyPluginsReachNoApp() {
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

/** A plugin takes components from `@numen/ui` and never reaches the window shell or the plugin loader. */
export function verifyPluginsReachNoShell() {
  const crossings = [
    ...findCrossings("plugins", "libs/ui/shell", ["['\"]@numen/shell\\b"]),
    ...findCrossings("plugins", "libs/plugin-host-ts", ["['\"]@numen/plugin-host\\b"]),
  ];
  if (crossings.length > 0) {
    throw new Error("A plugin reaches @numen/shell or @numen/plugin-host:\n" + crossings.join("\n"));
  }
}

/** What draws the notices of a window, which the window owns and a plugin reaches through `ctx.notices`. */
const NOTICE_NAMES = new Set(["Notices", "Notice", "createNotice", "createNoticeQueue", "NoticeQueue"]);

/** Every import of a package by a plugin that names one of NOTICE_NAMES. */
function findPluginNoticeImports() {
  const listed = spawnSync("git", ["ls-files", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8" });
  const pattern = /^\s*import\s+(?:type\s+)?\{([^}]*)\}\s*from\s*(['"])([^'"]+)\2/gm;
  const found = [];
  for (const file of (listed.stdout || "").split("\n")) {
    if (!/\.(ts|vue)$/.test(file)) continue;
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    for (const match of text.matchAll(pattern)) {
      if (match[3].startsWith(".")) continue;
      const names = match[1].split(",").map((one) => one.trim().replace(/^type\s+/, "").split(/\s+as\s+/)[0]);
      for (const name of names.filter((one) => NOTICE_NAMES.has(one))) found.push(`${file}: ${name} from ${match[3]}`);
    }
  }
  return found;
}

/** A plugin shows notices through `ctx.notices`; the window draws them, in one list. */
export function verifyPluginsDrawNoNotices() {
  const found = findPluginNoticeImports();
  if (found.length > 0) {
    throw new Error("A plugin draws notices itself instead of showing them through ctx.notices:\n" + found.join("\n"));
  }
}

/** Every window id literal, window route literal and name of the window's service in a plugin. */
function findPluginWindowNames() {
  const service = /\b((?:New)?WindowService(?:Client)?)\b/g;
  const found = [];
  for (const file of listShippedFiles(["plugins"])) {
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (WINDOW_IDS.includes(value) || WINDOW_ROUTE_PREFIXES.some((p) => value.startsWith(p))) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
    for (const hit of findInCode(source.code, service)) found.push({ file, ...hit });
  }
  return found;
}

/** A plugin reaches the window through the SDK: it names no window and creates no `WindowService` client. */
export function verifyPluginsNameNoWindow() {
  failOnFindings("A plugin names a window, window route, or WindowService", findPluginWindowNames());
}

/** The calls of Go's `os` and `filepath` packages that read or write the file system in a plugin service. */
const OS_WRITES = /\b(os\.(?:WriteFile|Create|CreateTemp|OpenFile|Mkdir|MkdirAll|MkdirTemp|Rename|Remove|RemoveAll|Symlink|Link|Chmod|Chtimes|Truncate))\(/g;
const OS_VAULT_ACCESS = /\b(os\.(?:WriteFile|Create|OpenFile|Mkdir|MkdirAll|Rename|Remove|RemoveAll|ReadFile|Open|Stat|ReadDir)|filepath\.WalkDir)\(/g;

/** Every direct `os` vault read/write or `NUMEN_VAULT_PATH` lookup in a shipped Go file of a plugin. */
function findPluginDataWrites() {
  const found = [];
  for (const file of listShippedFiles(["plugins/*.go"])) {
    const { literals, code } = readSource(file);
    for (const { line, value } of literals) {
      if (value === "NUMEN_VAULT_PATH") {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
    const touchesDotNumen = literals.some(({ value }) => value === ".numen" || value.startsWith(".numen/"));
    const isServiceFile = /\/backend\/service\//.test(file);
    const pattern = isServiceFile ? OS_VAULT_ACCESS : touchesDotNumen ? OS_WRITES : null;
    if (!pattern) continue;
    for (const hit of findInCode(code, pattern)) found.push({ file, ...hit });
  }
  return found;
}

/** A plugin accesses vault files and `.numen/` through the host store, never with `os`. */
export function verifyPluginsWriteDataThroughHost() {
  failOnFindings("A plugin accesses vault or .numen/ directly with os or NUMEN_VAULT_PATH", findPluginDataWrites());
}

/** Every reference a plugin makes to `libs/protocol`, `numen/v1/*.proto`, `@numen/protocol` or `@numen/wire`. */
function findPluginProtocolAndWireLeaks() {
  const found = [];
  const listed = spawnSync("git", ["ls-files", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8" });
  const files = (listed.stdout || "").split("\n").filter((f) => f && !/\/frontend\/gen\/|\.pb\.go$|\.connect\.go$/.test(f));
  for (const file of files) {
    const abs = path.join(REPO_ROOT, file);
    if (!fs.existsSync(abs)) continue;
    const text = fs.readFileSync(abs, "utf8");
    if (path.basename(file) === "go.mod") {
      text.split("\n").forEach((line, idx) => {
        if (!line.trim().startsWith("//") && line.includes("github.com/jiva-studio/numen/libs/protocol")) {
          found.push({ file, line: idx + 1, symbol: line.trim() });
        }
      });
    } else if (file.endsWith(".go")) {
      const { imports } = getGoReferences(text);
      for (const { line, value } of imports) {
        if (value.startsWith("github.com/jiva-studio/numen/libs/protocol")) {
          found.push({ file, line, symbol: `import "${value}"` });
        }
      }
    } else if (file.endsWith(".proto")) {
      text.split("\n").forEach((line, idx) => {
        const m = line.match(/^\s*import\s+"(numen\/v1\/[^"]+)"/);
        if (m) found.push({ file, line: idx + 1, symbol: `import "${m[1]}"` });
      });
    } else if (SCRIPT_EXTENSIONS.has(path.extname(file)) && path.basename(file) !== "eslint.config.js") {
      const { literals } = readSource(file);
      for (const { line, value } of literals) {
        if (
          value === "@numen/protocol" ||
          value.startsWith("@numen/protocol/") ||
          value === "@numen/wire" ||
          value.startsWith("@numen/wire/") ||
          value.includes("libs/protocol") ||
          value.includes("libs/wire")
        ) {
          found.push({ file, line, symbol: JSON.stringify(value) });
        }
      }
    }
  }
  return found;
}

/** A plugin depends on the SDK and `@numen/ui`, and knows nothing of `libs/protocol` or `@numen/wire`. */
export function verifyPluginsNoProtocolOrWire() {
  failOnFindings("A plugin depends on libs/protocol, numen/v1/*.proto, @numen/protocol or @numen/wire", findPluginProtocolAndWireLeaks());
}

/**
 * Every tie in `plugins/` to the surrounding monorepo that prevents moving `plugins/<name>` into its
 * own repository.
 */
function findPluginStandaloneViolations() {
  const found = [];
  for (const shared of ["plugins/vite.shared.ts", "plugins/vite.styles.ts", "plugins/vitest.config.ts"]) {
    if (fs.existsSync(path.join(REPO_ROOT, shared))) {
      found.push({ file: shared, line: 1, symbol: "shared file outside plugin directory" });
    }
  }
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  const pluginNames = fs
    .readdirSync(pluginsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith(".") && e.name !== "node_modules")
    .map((e) => e.name);
  const listed = spawnSync("git", ["ls-files", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8" });
  for (const file of (listed.stdout || "").split("\n").filter(Boolean)) {
    const parts = file.split("/");
    const plugin = parts[1];
    if (!pluginNames.includes(plugin)) continue;
    const pluginRoot = path.join(REPO_ROOT, "plugins", plugin);
    const abs = path.join(REPO_ROOT, file);
    if (!fs.existsSync(abs)) continue;
    const base = path.basename(file);
    if (base.endsWith(".proto") || base === "buf.gen.yaml") {
      found.push({ file, line: 1, symbol: `${base} (plugins communicate via SDK without own protobuf schema)` });
      continue;
    }
    if (/\/frontend\/gen\/|\.pb\.go$|\.connect\.go$/.test(file)) continue;
    const text = fs.readFileSync(abs, "utf8");
    if (base === "go.mod") {
      text.split("\n").forEach((line, idx) => {
        const trimmed = line.trim();
        if (trimmed.startsWith("//")) return;
        if (/=>\s*\.\.\//.test(trimmed)) {
          found.push({ file, line: idx + 1, symbol: `relative replace in go.mod: ${trimmed}` });
        }
        if (trimmed.includes("github.com/jiva-studio/numen/libs/notes") && !/=>\s*\.\.\//.test(trimmed)) {
          found.push({ file, line: idx + 1, symbol: `direct libs/notes requirement: ${trimmed}` });
        }
      });
      continue;
    }
    if (file.endsWith(".go")) {
      const { imports, literals } = getGoReferences(text);
      for (const { line, value } of imports) {
        if (value.startsWith("github.com/jiva-studio/numen/libs/notes")) {
          found.push({ file, line, symbol: `import "${value}"` });
        }
      }
      for (const { line, value } of literals) {
        if (/^\.\.\//.test(value) && !isWithin(path.resolve(path.dirname(abs), value), pluginRoot)) {
          found.push({ file, line, symbol: `relative path escapes plugins/${plugin}: ${JSON.stringify(value)}` });
        }
      }
      continue;
    }
    if (base === "eslint.config.js") {
      for (const specifier of getImportSpecifiers(text)) {
        if (specifier.startsWith(".") && !isWithin(path.resolve(path.dirname(abs), specifier), pluginRoot)) {
          found.push({ file, line: 1, symbol: `relative import escapes plugins/${plugin}: ${JSON.stringify(specifier)}` });
        }
      }
      continue;
    }
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      const cleanVal = value.split("?")[0];
      if (/^\.\.\//.test(cleanVal) && !isWithin(path.resolve(path.dirname(abs), cleanVal), pluginRoot)) {
        found.push({ file, line, symbol: `relative path escapes plugins/${plugin}: ${JSON.stringify(value)}` });
      }
    }
  }
  return found;
}

/** Every plugin directory `plugins/<name>` is self-contained and movable to a standalone repository. */
export function verifyPluginsStandaloneRepo() {
  failOnFindings("A plugin is tied to the monorepo by relative paths, libs/notes, or its own .proto", findPluginStandaloneViolations());
}

/** The tab kinds, text kinds and command/menu ids owned by specific plugins. */
const PLUGIN_OWNED_TOKENS = [
  { token: "note", owners: ["notes"] },
  { token: "plex", owners: ["views-plex-graph"] },
  { token: "document", owners: ["readers-pdf"] },
  { token: "book", owners: ["readers-epub"] },
  { token: "recording", owners: ["views-media-player"] },
  { token: "deck", owners: ["views-flashcards-studio"] },
  { token: "card-model", owners: ["views-flashcards-studio"] },
  { token: "preset", owners: ["views-flashcards-studio"] },
  { token: "transcript", owners: ["engines-transcription", "views-media-player"] },
  { token: "recognition", owners: ["engines-ocr", "readers-pdf"] },
  { token: "page-text", owners: ["readers-pdf"] },
  { token: "transcribe", owners: ["engines-transcription"] },
  { token: "proofread", owners: ["tools-proofreading"] },
  { token: "recognise", owners: ["engines-ocr"] },
  { token: "relation", owners: ["views-plex-graph"] },
];

/** Every reference in a plugin to another plugin's manifest id, tab kind, text kind or menu/command id. */
function findCrossPluginKnowledge() {
  const pluginsDir = path.join(MODULES_DIR, "plugins");
  const pluginNames = fs
    .readdirSync(pluginsDir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith(".") && e.name !== "node_modules")
    .map((e) => e.name);
  const manifestIds = new Map();
  for (const plugin of pluginNames) {
    const mPath = path.join(pluginsDir, plugin, "manifest.json");
    if (fs.existsSync(mPath)) {
      const m = JSON.parse(fs.readFileSync(mPath, "utf8"));
      if (m.id) manifestIds.set(m.id, plugin);
    }
  }
  const found = [];
  const listed = spawnSync("git", ["ls-files", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8" });
  for (const file of (listed.stdout || "").split("\n").filter(Boolean)) {
    if (/\/frontend\/gen\/|\.pb\.go$|\.connect\.go$/.test(file) || path.basename(file) === "eslint.config.js") continue;
    const plugin = file.split("/")[1];
    if (!pluginNames.includes(plugin)) continue;
    if (path.basename(file) === "manifest.json") {
      const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
      text.split("\n").forEach((line, idx) => {
        for (const { token, owners } of PLUGIN_OWNED_TOKENS) {
          if (!owners.includes(plugin) && new RegExp(`"${token}"`).test(line)) {
            found.push({ file, line: idx + 1, symbol: `"${token}" (owned by ${owners.join(", ")})` });
          }
        }
      });
      continue;
    }
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      const idOwner = manifestIds.get(value);
      if (idOwner && idOwner !== plugin) {
        found.push({ file, line, symbol: `${JSON.stringify(value)} (plugin ID of ${idOwner})` });
      }
      if (!NOT_SHIPPED.test(file)) {
        for (const { token, owners } of PLUGIN_OWNED_TOKENS) {
          if (!owners.includes(plugin) && value === token) {
            found.push({ file, line, symbol: `${JSON.stringify(value)} (owned by ${owners.join(", ")})` });
          }
        }
      }
    }
  }
  return found;
}

/** A plugin knows no other plugin's ID, tab kind, text kind or command ID. */
export function verifyPluginsNoCrossPluginKnowledge() {
  failOnFindings("A plugin hardcodes another plugin's ID, tab kind, text kind or command", findCrossPluginKnowledge());
}

/** G10: Plugin backend internal layers (inner packages under backend/<pkg> must not import backend/service or backend/engine). */
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
  failOnFindings("Plugin backend inner package imports outer service/engine package (G10)", findPluginBackendLayerInversions());
}

/**
 * P4 / G14: Golden Template & Config Drift Verification for Standalone Plugins.
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
    if (getImportSpecifiers(exText).some((s) => s.startsWith(".")) || exText.includes("@numen/plugin-sdk/eslint")) {
      found.push({ file: "plugins/.example/eslint.config.js", line: 1, symbol: "plugins/.example/eslint.config.js must be 100% self-contained without ../ paths or SDK coupling" });
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
  failOnFindings("Plugin configuration missing or drifted from golden template plugins/.example/ (P4)", findPluginConfigDrift());
}

/**
 * The application compiles nothing from `plugins/`, and a plugin compiles nothing from `apps/`.
 * The window reaches a plugin only through `/plugin/<id>/` at runtime.
 */
export function verifyAppPluginBoundary() {
  const findings = [
    ...findBoundaryReferences("apps", "plugins", /^@plugins(\/|$)/, getPackageNames("plugins")),
    ...findBoundaryReferences("plugins", "apps", /^@(\/|$)/, getPackageNames("apps")),
  ];
  if (findings.length > 0) {
    throw new Error("Application and plugins reach into each other:\n    " + findings.join("\n    "));
  }
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
    name: "Plugin Configs Match Golden Example plugins/.example/ & Root .golangci.yml (P4)",
    run: verifyPluginConfigGoldenParity,
  },
  {
    id: "plugin-backend-layers",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Backend Internal Hexagonal Layer Direction (G10)",
    run: verifyPluginBackendInternalLayers,
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
    id: "plugins-reach-no-shell",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Reach No Shell or Plugin Host (no @numen/shell, no @numen/plugin-host, no path into either)",
    run: verifyPluginsReachNoShell,
  },
  {
    id: "plugins-draw-no-notices",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Draw No Notices (no Notices, Notice, createNotice or notice queue imported; ctx.notices instead)",
    run: verifyPluginsDrawNoNotices,
  },
  {
    id: "plugins-name-no-window",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Name No Window (no window ids, no window routes, no WindowService client)",
    run: verifyPluginsNameNoWindow,
  },
  {
    id: "plugins-write-data-through-host",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Access Vault and .numen/ Only Through the Host Store (no direct os fs or NUMEN_VAULT_PATH)",
    run: verifyPluginsWriteDataThroughHost,
  },
  {
    id: "plugins-no-protocol-or-wire",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Depend Only on SDK (no libs/protocol, numen/v1/*.proto, @numen/protocol or @numen/wire)",
    run: verifyPluginsNoProtocolOrWire,
  },
  {
    id: "plugins-no-cross-plugin-knowledge",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Know Nothing of Other Plugins (no cross-plugin IDs, tab kinds, text kinds or command IDs)",
    run: verifyPluginsNoCrossPluginKnowledge,
  },
  {
    id: "plugins-standalone-repo",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Are Standalone-Repo Ready (no ../ escapes, no vite.shared, no go.mod replace ../, no libs/notes, no plugin .proto)",
    run: verifyPluginsStandaloneRepo,
  },
  {
    id: "apps-plugins-boundary",
    scope: "plugins",
    module: "boundary",
    name: "Application & Plugin Separation (apps compile nothing from plugins, plugins nothing from apps)",
    run: verifyAppPluginBoundary,
  },
  {
    id: "plugins-frontend",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Frontend Contract & Boundaries (no @/**, no @plugins/**)",
    run: verifyPluginsFrontend,
  },
];
