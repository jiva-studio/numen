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
 *
 * Options:
 *   --update-baseline  remove baseline.json entries that no longer occur
 *   --no-baseline      list every violation of the checks that read baseline.json
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

  // 5. Verify no plugin frontend imports another plugin's files or package
  const frontendLeaks = findCrossPluginImports(pluginNames);
  if (frontendLeaks.length > 0) {
    throw new Error("Cross-plugin import detected:\n" + frontendLeaks.join("\n"));
  }
}

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

/** A plugin takes components from `@numen/ui` and never reaches the window shell or the plugin loader. */
function verifyPluginsReachNoShell() {
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
function verifyPluginsDrawNoNotices() {
  const found = findPluginNoticeImports();
  if (found.length > 0) {
    throw new Error("A plugin draws notices itself instead of showing them through ctx.notices:\n" + found.join("\n"));
  }
}

/** The packages of the core that belong to a plugin, as directories, and as files of `container`. */
const CORE_MOVED_DIRS = [
  "usecase/document",
  "usecase/recognition",
  "usecase/transcription",
  "usecase/embedding",
  "internal/embedding",
  "internal/ocrartifact",
  "internal/transcript",
  "port/transcriptionport",
  "internal/adapter/plugin/transcriberbridge",
  "internal/adapter/plugin/proofreaderbridge",
  "internal/urlfile",
  "internal/adapter/download",
  "internal/adapter/plugin/downloaderbridge",
  "port/downloadport",
  "internal/correction",
  "adapter/window/flashcards",
  "internal/adapter/plugin/toolsbridge",
  "usecase/plugintool",
];
const CORE_MOVED_CONTAINER_FILES = ["recogniser", "transcriber", "proofreader", "embedder", "downloader"];

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
  const container = listFiles("libs/core/container").filter((f) => path.dirname(f) === "libs/core/container");
  for (const name of CORE_MOVED_CONTAINER_FILES) {
    const files = container.filter((f) => new RegExp(`/${name}(_\\w+)?\\.go$`).test(f));
    if (files.length > 0) found.push(`libs/core/container/${name}*.go: ${files.map((f) => path.basename(f)).join(", ")}`);
  }
  if (found.length > 0) {
    throw new Error("The core holds packages a plugin owns:\n" + found.join("\n"));
  }
}

/** The Go module path every module of the repository is named under. */
const GO_MODULE_ROOT = "github.com/jiva-studio/numen/";

/** The source files read for references, by extension. */
const SCRIPT_EXTENSIONS = new Set([".ts", ".tsx", ".mts", ".cts", ".js", ".mjs", ".cjs", ".vue"]);

/** Lists every string literal of a script with its line, comments left out. */
function getScriptLiterals(source) {
  return readScript(source).literals;
}

/**
 * Lists every string literal of a script with its line, and returns the script with its comments,
 * regular expressions and literals blanked so lines keep their numbers. A template literal is read
 * up to its first substitution.
 */
function readScript(source) {
  const literals = [];
  const blanked = [];
  let line = 1;
  let i = 0;
  let previous = "";
  while (i < source.length) {
    const c = source[i];
    const next = source[i + 1];
    if (c === "\n") {
      line++;
      i++;
      continue;
    }
    const from = i;
    if (c === "/" && next === "/") {
      while (i < source.length && source[i] !== "\n") i++;
      blanked.push([from, i]);
      continue;
    }
    if (c === "/" && next === "*") {
      i += 2;
      while (i < source.length && !(source[i] === "*" && source[i + 1] === "/")) {
        if (source[i] === "\n") line++;
        i++;
      }
      i += 2;
      blanked.push([from, i]);
      continue;
    }
    if (c === "/" && (previous === "" || "(,=:[!&|?{};+-*%<>~^".includes(previous))) {
      i++;
      let inClass = false;
      while (i < source.length && source[i] !== "\n") {
        if (source[i] === "\\") i++;
        else if (source[i] === "[") inClass = true;
        else if (source[i] === "]") inClass = false;
        else if (source[i] === "/" && !inClass) break;
        i++;
      }
      i++;
      previous = "/";
      blanked.push([from, i]);
      continue;
    }
    if (c === "'" || c === '"' || c === "`") {
      const start = line;
      let value = "";
      let isOpen = true;
      i++;
      while (i < source.length && source[i] !== c) {
        if (source[i] === "\\") {
          value += source[i + 1] ?? "";
          i += 2;
          continue;
        }
        if (c === "`" && source[i] === "$" && source[i + 1] === "{") isOpen = false;
        if (source[i] === "\n") line++;
        if (isOpen) value += source[i];
        i++;
      }
      i++;
      literals.push({ line: start, value });
      previous = "'";
      blanked.push([from, i]);
      continue;
    }
    if (!/\s/.test(c)) previous = c;
    i++;
  }
  let code = "";
  let at = 0;
  for (const [from, to] of blanked) {
    code += source.slice(at, from) + source.slice(from, to).replace(/[^\n]/g, " ");
    at = to;
  }
  return { literals, code: code + source.slice(at) };
}

/** The `<script>` blocks of a single-file component, blanked elsewhere so lines keep their numbers. */
function getVueScript(source) {
  return source.replace(/(^|<\/script>)([\s\S]*?)(<script\b[^>]*>|$)/g, (_, open, between, close) =>
    open + between.replace(/[^\n]/g, " ") + close.replace(/[^\n]/g, " "),
  );
}

/** Lists the import paths of a Go file with their lines, and its other string literals. */
function getGoReferences(source) {
  const imports = [];
  const literals = [];
  const stripped = source.replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, " "));
  const lines = stripped.split("\n");
  let inImportBlock = false;
  lines.forEach((text, index) => {
    const code = text.replace(/\/\/.*$/, "");
    const line = index + 1;
    if (/^import\s*\($/.test(code.trim())) {
      inImportBlock = true;
      return;
    }
    if (inImportBlock && code.trim() === ")") {
      inImportBlock = false;
      return;
    }
    const single = /^import\s+(?:[\w.]+\s+)?"([^"]+)"/.exec(code.trim());
    if (single) {
      imports.push({ line, value: single[1] });
      return;
    }
    if (inImportBlock) {
      const spec = /"([^"]+)"/.exec(code);
      if (spec) imports.push({ line, value: spec[1] });
      return;
    }
    for (const match of code.matchAll(/"((?:[^"\\]|\\.)*)"|`([^`]*)`/g)) {
      literals.push({ line, value: match[1] ?? match[2] });
    }
  });
  return { imports, literals };
}

/** Lists the module paths and local replacement paths a `go.mod` names. */
function getGoModReferences(source) {
  const imports = [];
  const literals = [];
  source.split("\n").forEach((text, index) => {
    const line = index + 1;
    const code = text.replace(/\/\/.*$/, "");
    if (/^\s*module\s/.test(code)) return;
    for (const match of code.matchAll(/github\.com\/jiva-studio\/numen\/[\w./-]+/g)) {
      imports.push({ line, value: match[0] });
    }
    const replacement = /=>\s*(\.{1,2}\/\S*)/.exec(code);
    if (replacement) literals.push({ line, value: replacement[1] });
  });
  return { imports, literals };
}

/** True when `target` is `root` or a path below it. */
function isWithin(target, root) {
  const relative = path.relative(root, target);
  return relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));
}

/**
 * Lists every reference a file under `fromDir` makes into `toDir`.
 * A reference is an import path, an alias, a relative file path resolved
 * against the file's directory, or a repository-rooted path.
 */
function findBoundaryReferences(fromDir, toDir, aliasPattern, packageNames) {
  const toRoot = path.join(REPO_ROOT, toDir);
  const goPrefix = GO_MODULE_ROOT + toDir + "/";
  const listed = spawnSync("git", ["ls-files", "-z", "--", fromDir], { cwd: REPO_ROOT, encoding: "utf8" });
  const files = (listed.stdout || "").split("\0").filter((file) => file && !BOUNDARY_RULES.has(file));
  const findings = [];
  for (const file of files) {
    const absolute = path.join(REPO_ROOT, file);
    if (!fs.existsSync(absolute)) continue;
    const base = path.basename(file);
    const ext = path.extname(file);
    const directory = path.dirname(absolute);
    const report = (line, message) => findings.push(file + ":" + line + ": " + message);
    const checkPath = ({ line, value }) => {
      if (aliasPattern.test(value)) {
        report(line, "alias " + JSON.stringify(value));
      } else if (/^\.{1,2}\//.test(value) && isWithin(path.resolve(directory, value), toRoot)) {
        report(line, "path " + JSON.stringify(value) + " reaches " + toDir + "/");
      } else if (value.startsWith(toDir + "/")) {
        report(line, "repository path " + JSON.stringify(value));
      } else if (value.startsWith(goPrefix)) {
        report(line, "module " + JSON.stringify(value));
      }
    };
    const source = () => fs.readFileSync(absolute, "utf8");
    if (base === "go.mod" || ext === ".go") {
      const { imports, literals } = base === "go.mod" ? getGoModReferences(source()) : getGoReferences(source());
      for (const { line, value } of imports) {
        if (value.startsWith(goPrefix)) report(line, "Go import " + JSON.stringify(value));
      }
      literals.forEach(checkPath);
    } else if (SCRIPT_EXTENSIONS.has(ext)) {
      const text = ext === ".vue" ? getVueScript(source()) : source();
      getScriptLiterals(text).forEach(checkPath);
    } else if (/^tsconfig.*\.json$/.test(base)) {
      getScriptLiterals(source()).forEach(checkPath);
    } else if (base === "package.json") {
      const manifest = JSON.parse(source());
      for (const field of ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"]) {
        for (const name of Object.keys(manifest[field] ?? {})) {
          if (packageNames.has(name)) report(1, field + " names " + JSON.stringify(name));
        }
      }
    }
  }
  return findings;
}

/** The npm package names declared under a top-level directory. */
function getPackageNames(dir) {
  const listed = spawnSync("git", ["ls-files", "--", dir + "/**/package.json", dir + "/*/package.json"], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  });
  const names = new Set();
  for (const file of (listed.stdout || "").split("\n").filter(Boolean)) {
    const name = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, file), "utf8")).name;
    if (typeof name === "string") names.add(name);
  }
  return names;
}

/**
 * The application compiles nothing from `plugins/`, and a plugin compiles nothing from `apps/`.
 * The window reaches a plugin only through `/plugin/<id>/` at runtime.
 */
function verifyAppPluginBoundary() {
  const findings = [
    ...findBoundaryReferences("apps", "plugins", /^@plugins(\/|$)/, getPackageNames("plugins")),
    ...findBoundaryReferences("plugins", "apps", /^@(\/|$)/, getPackageNames("apps")),
  ];
  if (findings.length > 0) {
    throw new Error("Application and plugins reach into each other:\n    " + findings.join("\n    "));
  }
}

/**
 * The protos of the domains plugins own. Every one generates into the shared `numen/v1` package.
 * `recording.proto` (the core serves the address a file plays at) and `embedding.proto` (the core
 * asks the embeddings plugin for vectors and keeps them in its index) are the core's own contracts.
 */
const DOMAIN_PROTOS = [
  "transcript",
  "ocr",
  "article",
  "artifact",
  "cards",
  "presets",
  "flashcards",
  "transcription",
  "download",
  "proofreading",
];
const PROTOCOL_GEN = "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1";

/** The exported top-level names a generated Go file declares. */
function readGoExports(file) {
  if (!fs.existsSync(file)) return [];
  const names = [];
  let block = "";
  for (const line of fs.readFileSync(file, "utf8").split("\n")) {
    const open = line.match(/^(const|var) \($/);
    if (open) { block = open[1]; continue; }
    if (block && line === ")") { block = ""; continue; }
    const member = block && line.match(/^\t([A-Z]\w*)\b/);
    if (member) names.push(member[1]);
    const top = line.match(/^(?:type|func|const|var) ([A-Z]\w*)/);
    if (top) names.push(top[1]);
  }
  return names;
}

/** Which domain proto each generated name comes from, by package: `v1` and `numenv1connect`. */
function readDomainSymbols() {
  const gen = path.join(REPO_ROOT, "libs/protocol/gen/numen/v1");
  const symbols = { [PROTOCOL_GEN]: new Map(), [PROTOCOL_GEN + "/numenv1connect"]: new Map() };
  for (const proto of DOMAIN_PROTOS) {
    for (const name of readGoExports(path.join(gen, proto + ".pb.go"))) {
      symbols[PROTOCOL_GEN].set(name, proto);
    }
    for (const name of readGoExports(path.join(gen, "numenv1connect", proto + ".connect.go"))) {
      symbols[PROTOCOL_GEN + "/numenv1connect"].set(name, proto);
    }
  }
  return symbols;
}

/** The core names no message, enum or service of a domain proto. */
function verifyCoreImportsNoDomainProto() {
  const symbols = readDomainSymbols();
  const files = spawnSync("git", ["ls-files", "--", "libs/core/*.go"], { cwd: REPO_ROOT, encoding: "utf8" })
    .stdout.split("\n")
    .filter((f) => f && fs.existsSync(path.join(REPO_ROOT, f)));
  const found = [];
  const byProto = new Map();
  for (const file of files) {
    const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    const aliases = new Map();
    for (const [, alias, pkg] of text.matchAll(/^\s*(?:import\s+)?(\w+\s+)?"([^"]+)"\s*$/gm)) {
      if (!symbols[pkg]) continue;
      aliases.set(alias ? alias.trim() : path.basename(pkg) === "v1" ? "numenv1" : path.basename(pkg), symbols[pkg]);
    }
    if (aliases.size === 0) continue;
    text.split("\n").forEach((line, i) => {
      const hits = new Set();
      for (const [, alias, name] of line.matchAll(/\b(\w+)\.([A-Z]\w*)\b/g)) {
        const proto = aliases.get(alias)?.get(name);
        if (!proto) continue;
        hits.add(`${name} (${proto}.proto)`);
        byProto.set(proto, (byProto.get(proto) ?? 0) + 1);
      }
      if (hits.size > 0) found.push(`${file}:${i + 1}: ${[...hits].join(", ")}`);
    });
  }
  if (found.length > 0) {
    const summary = [...byProto].sort().map(([proto, count]) => `  ${proto}.proto: ${count}`);
    throw new Error(
      "The core names domain protos:\n" + summary.join("\n") + "\n" + found.join("\n"),
    );
  }
}

/** The tab kinds, note types and commands plugins own. */
const PLUGIN_KINDS = ["deck", "card-model", "preset", "plex", "agent", "files"];

/**
 * Files of the window where one of PLUGIN_KINDS names something of the core: `agent` as the type
 * and path of a setting, `files` and `agent` as services of the host a plugin is handed.
 */
const CORE_NAME_FILES = [
  "apps/desktop/editor/src/entities/settings/",
  "apps/desktop/editor/src/pages/settings/",
  "apps/desktop/editor/src/entities/plugin/model/context.ts",
];

/** The window names no tab kind, note type or command of a plugin as a literal. */
function verifyWindowNamesNoPluginKind() {
  const literal = `['"\`](${PLUGIN_KINDS.join("|")})['"\`]`;
  const grep = spawnSync(
    "git",
    [
      "grep", "-nE", literal, "--",
      "apps/desktop/editor/src/*.ts", "apps/desktop/editor/src/*.vue",
      ":!*.test.ts", ":!*.spec.ts", ":!*.stories.ts",
    ],
    { cwd: REPO_ROOT, encoding: "utf8" },
  );
  const found = (grep.stdout || "")
    .split("\n")
    .filter(Boolean)
    .filter((line) => !CORE_NAME_FILES.some((prefix) => line.split(":")[0].startsWith(prefix)));
  if (found.length > 0) {
    throw new Error("The window names plugin kinds:\n" + found.join("\n"));
  }
}

/** The texts made from a file and the files a plugin keeps, named by the plugin that makes them. */
const PRODUCERS = ["asr", "ocr", "captions", "article", "copy", "review-log"];

/** The notes library names no producer of a text or of a plugin's file. */
function verifyNotesNamesNoProducer() {
  const grep = spawnSync(
    "git",
    ["grep", "-nE", `"(${PRODUCERS.join("|")})"`, "--", "libs/notes/*.go", ":!*_test.go"],
    { cwd: REPO_ROOT, encoding: "utf8" },
  );
  const found = (grep.stdout || "").split("\n").filter(Boolean);
  if (found.length > 0) {
    throw new Error("The notes library names producers:\n" + found.join("\n"));
  }
}

/**
 * The views plugins own, which `@numen/ui` neither holds nor exports: the directories they were
 * moved out of `libs/ui/core/src` from, and the names `src/index.ts` exported them by.
 */
const PLUGIN_VIEWS = [
  {
    plugin: "views-agent-chat",
    dirs: ["screens/agent", "features/thread", "features/message-composer"],
    names: ["Agent", "Thread", "MessageComposer", "useConversation", "Conversation", "Turn", "AgentModelOption", "AgentPort", "AgentStep"],
  },
];

/** The names `libs/ui/core/src/index.ts` exports in braces. */
function getUiExports() {
  const index = fs.readFileSync(path.join(MODULES_DIR, "libs/ui/core/src/index.ts"), "utf8");
  return Array.from(index.matchAll(/^export\s+(?:type\s+)?\{([^}]*)\}/gm)).flatMap((match) =>
    match[1]
      .split(",")
      .map((one) => one.trim().replace(/^type\s+/, "").split(/\s+as\s+/).pop())
      .filter(Boolean),
  );
}

/** `@numen/ui` holds and exports none of the views a plugin owns. */
function verifyUiHoldsNoPluginView() {
  const exported = new Set(getUiExports());
  const found = [];
  for (const { plugin, dirs, names } of PLUGIN_VIEWS) {
    for (const dir of dirs) {
      if (fs.existsSync(path.join(MODULES_DIR, "libs/ui/core/src", dir))) {
        found.push(`libs/ui/core/src/${dir} belongs to ${plugin}`);
      }
    }
    for (const name of names) {
      if (exported.has(name)) found.push(`@numen/ui exports ${name}, which belongs to ${plugin}`);
    }
  }
  if (found.length > 0) {
    throw new Error("The UI library holds views plugins own:\n" + found.join("\n"));
  }
}

/** The window chrome `@numen/shell` holds: its directories under `src/` and the names it exports. */
const SHELL_PARTS = {
  dirs: ["features/workspace", "features/palette", "features/status-bar", "features/notices", "features/welcome", "screens"],
  names: ["WorkspaceLayout", "Palette", "StatusBar", "Notices", "createNoticeQueue", "WelcomePage", "getVaultForKey", "VaultRow", "WelcomeAction"],
};

/** `@numen/ui` holds and exports none of the window chrome `@numen/shell` owns. */
function verifyUiHoldsNoShellPart() {
  const exported = new Set(getUiExports());
  const found = [
    ...SHELL_PARTS.dirs
      .filter((dir) => fs.existsSync(path.join(MODULES_DIR, "libs/ui/core/src", dir)))
      .map((dir) => `libs/ui/core/src/${dir} belongs to libs/ui/shell`),
    ...SHELL_PARTS.names.filter((name) => exported.has(name)).map((name) => `@numen/ui exports ${name}, which belongs to @numen/shell`),
  ];
  if (found.length > 0) {
    throw new Error("The UI library holds the window chrome:\n" + found.join("\n"));
  }
}

function verifyUiDomainIsolation() {
  const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire|shell)|apps|libs/ui/shell)", "libs/ui/core/src/"], { cwd: REPO_ROOT });
  const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
  if (leaks) {
    throw new Error("UI library imports domain, wire, protocol, the shell or apps:\n" + leaks);
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/ui/core"));
}

function verifyShellDomainIsolation() {
  const leakCheck = spawnSync("git", ["grep", "-E", "(@numen/(protocol|wire)|apps)", "libs/ui/shell/src/"], { cwd: REPO_ROOT });
  const leaks = (leakCheck.stdout ? leakCheck.stdout.toString() : "").trim();
  if (leaks) {
    throw new Error("Shell library imports domain, wire, protocol or apps:\n" + leaks);
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/ui/shell"));
}

/**
 * What `libs/plugin-host-ts/src` may import besides its own files: Vue and the plugin SDK; its
 * tests also vitest, and `vite.ts`, the build of the host modules, also vite.
 */
const PLUGIN_HOST_IMPORTS = [
  { files: /./, specifiers: /^(vue|@numen\/plugin-sdk)(\/|$)/ },
  { files: /\.test\.ts$/, specifiers: /^vitest$/ },
  { files: /^vite\.ts$/, specifiers: /^vite$/ },
];

/**
 * The module specifiers a TypeScript source names in an `import` or `export … from` statement and
 * in `import()`. A statement starts its line, so the source of a module a string holds is not read.
 */
function getImportSpecifiers(source) {
  const pattern =
    /^\s*(?:(?:import|export)\s[^'"();=]*?\bfrom|import)\s*(['"])([^'"]+)\1|\bimport\s*\(\s*(['"])([^'"]+)\3/gm;
  return Array.from(source.matchAll(pattern), (match) => match[2] ?? match[4]);
}

/** Every import of `libs/plugin-host-ts/src` that is neither relative nor in PLUGIN_HOST_IMPORTS. */
function findPluginHostImports() {
  const src = path.join(MODULES_DIR, "libs/plugin-host-ts/src");
  const found = [];
  for (const file of fs.readdirSync(src, { recursive: true })) {
    if (!/\.(ts|vue)$/.test(file)) continue;
    const text = fs.readFileSync(path.join(src, file), "utf8");
    for (const specifier of getImportSpecifiers(text)) {
      if (specifier.startsWith(".")) continue;
      const allowed = PLUGIN_HOST_IMPORTS.some(
        (rule) => rule.files.test(path.basename(file)) && rule.specifiers.test(specifier),
      );
      if (!allowed) found.push(`libs/plugin-host-ts/src/${file}: ${specifier}`);
    }
  }
  return found;
}

/** The plugin loader knows Vue and the plugin SDK, and nothing of the UI, the shell, the wire or an application. */
function verifyPluginHostIsolation() {
  const found = findPluginHostImports();
  if (found.length > 0) {
    throw new Error("The plugin host imports beyond vue and @numen/plugin-sdk:\n" + found.join("\n"));
  }
  exec("npm run lint", path.join(MODULES_DIR, "libs/plugin-host-ts"));
}

/**
 * Lists every string literal of a Go or proto source with its line, and returns the source with its
 * comments and literals blanked so lines keep their numbers.
 */
function readGo(source) {
  const literals = [];
  let code = "";
  let line = 1;
  let i = 0;
  const blank = (text) => text.replace(/[^\n]/g, " ");
  while (i < source.length) {
    const c = source[i];
    const from = i;
    if (c === "/" && source[i + 1] === "/") {
      while (i < source.length && source[i] !== "\n") i++;
      code += blank(source.slice(from, i));
      continue;
    }
    if (c === "/" && source[i + 1] === "*") {
      const end = source.indexOf("*/", i + 2);
      i = end < 0 ? source.length : end + 2;
      const text = source.slice(from, i);
      line += text.split("\n").length - 1;
      code += blank(text);
      continue;
    }
    if (c === '"' || c === "'" || c === "`") {
      let value = "";
      i++;
      while (i < source.length && source[i] !== c && (c === "`" || source[i] !== "\n")) {
        if (c !== "`" && source[i] === "\\") {
          value += source[i + 1] ?? "";
          i += 2;
          continue;
        }
        value += source[i];
        i++;
      }
      i++;
      literals.push({ line, value });
      const text = source.slice(from, i);
      line += text.split("\n").length - 1;
      code += blank(text);
      continue;
    }
    if (c === "\n") line++;
    code += c;
    i++;
  }
  return { literals, code };
}

/** Reads a tracked source file as literals and code, by its extension; null for any other file. */
function readSource(file) {
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  if (/\.(go|proto)$/.test(file)) return readGo(text);
  if (file.endsWith(".vue")) return readScript(getVueScript(text));
  if (/\.(ts|tsx|mts|cts|js|mjs|cjs)$/.test(file)) return readScript(text);
  return null;
}

/** Every match of `pattern` in the code of a source, with its line and the text it matched. */
function findInCode(code, pattern) {
  const found = [];
  code.split("\n").forEach((text, index) => {
    for (const match of text.matchAll(pattern)) found.push({ line: index + 1, symbol: match[1] ?? match[0] });
  });
  return found;
}

/** Tests, stories, test kits of a window, fixtures and generated code. */
const NOT_SHIPPED = /(_test\.go|\.(test|spec|stories)\.ts|\.pb\.go|\.connect\.go)$|(^|\/)(testing|testdata|e2e|gen)\//;

/** The tracked files under `pathspecs` that ship: no test, story, fixture or generated file. */
function listShippedFiles(pathspecs) {
  const listed = spawnSync("git", ["ls-files", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && !NOT_SHIPPED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
}

/**
 * The known violations of the checks that read one, by check id, as `<file>: <symbol>` and the
 * number of times it occurs there. Line numbers are left out so an edit above does not move them.
 */
const BASELINE_FILE = path.join(SCRIPT_DIR, "baseline.json");
const baseline = fs.existsSync(BASELINE_FILE) ? JSON.parse(fs.readFileSync(BASELINE_FILE, "utf8")) : {};
let isBaselineChanged = false;

/**
 * Fails on a finding the baseline does not hold, and on a baseline entry found fewer times than it
 * lists. `--update-baseline` lowers the listed counts to what is found and never raises one;
 * `--no-baseline` fails on every finding.
 */
function checkAgainstBaseline(id, title, findings) {
  const counts = new Map();
  for (const { file, symbol } of findings) {
    const key = `${file}: ${symbol}`;
    counts.set(key, (counts.get(key) ?? 0) + 1);
  }
  if (flags.has("--no-baseline")) {
    if (findings.length === 0) return;
    const listed = findings.map(({ file, line, symbol }) => `${file}:${line}: ${symbol}`);
    throw new Error(`${title} (${findings.length}):\n` + listed.join("\n"));
  }
  const known = baseline[id] ?? {};
  const added = [...counts]
    .filter(([key, count]) => count > (known[key] ?? 0))
    .map(([key, count]) => `${key} (found ${count}, baseline ${known[key] ?? 0})`);
  const gone = Object.entries(known)
    .filter(([key, count]) => (counts.get(key) ?? 0) < count)
    .map(([key, count]) => `${key} (found ${counts.get(key) ?? 0}, baseline ${count})`);
  if (flags.has("--update-baseline") && gone.length > 0) {
    const kept = Object.entries(known)
      .map(([key, count]) => [key, Math.min(count, counts.get(key) ?? 0)])
      .filter(([, count]) => count > 0);
    baseline[id] = Object.fromEntries(kept);
    isBaselineChanged = true;
    gone.length = 0;
  }
  const problems = [];
  if (added.length > 0) problems.push(`${title}, not in ${path.basename(BASELINE_FILE)}:\n` + added.join("\n"));
  if (gone.length > 0) {
    problems.push(
      `Baseline entries of ${id} that no longer occur; run node check.mjs --update-baseline ${id}:\n` + gone.join("\n"),
    );
  }
  if (problems.length > 0) throw new Error(problems.join("\n"));
}

/** Writes the baseline back when `--update-baseline` lowered it. */
function writeBaseline() {
  if (!isBaselineChanged) return;
  const sorted = Object.fromEntries(
    Object.keys(baseline)
      .sort()
      .filter((id) => Object.keys(baseline[id]).length > 0)
      .map((id) => [id, Object.fromEntries(Object.entries(baseline[id]).sort(([a], [b]) => a.localeCompare(b)))]),
  );
  fs.writeFileSync(BASELINE_FILE, JSON.stringify(sorted, null, 2) + "\n");
  console.log("Baseline written: " + path.relative(REPO_ROOT, BASELINE_FILE));
}

/**
 * The services, enums, values, messages and calls of `numen.v1` that a single plugin owns. Each moves
 * into the plugin's own proto package or gives way to a shared mechanism.
 */
const PLUGIN_PROTO_SYMBOLS = [
  "TranscriptService",
  "ArticleService",
  "BookService",
  "DocumentService",
  "SOURCE_KIND_BOOK",
  "SOURCE_KIND_RECORDING",
  "SOURCE_KIND_URL",
  "BookFormat",
  "FILE_MENU_ACTION_RECOGNISE",
  "FILE_MENU_ACTION_PROOFREAD",
  "BookProgress",
  "RecordingProgress",
  "SetVaultInUse",
  "SetVaultInUseRequest",
  "SetVaultInUseResponse",
];

/** The fields of `agent.proto` that name a card of the flashcards plugin. */
const PLUGIN_AGENT_FIELDS = ["mark", "face"];

/** Every name of PLUGIN_PROTO_SYMBOLS and field of PLUGIN_AGENT_FIELDS in the code of `numen.v1`. */
function findPluginProtoSymbols() {
  const symbols = new RegExp(`\\b(${PLUGIN_PROTO_SYMBOLS.join("|")})\\b`, "g");
  const fields = new RegExp(`^\\s*(?:repeated\\s+|optional\\s+)?[\\w.]+\\s+(${PLUGIN_AGENT_FIELDS.join("|")})\\s*=\\s*\\d+`, "g");
  const found = [];
  for (const file of listShippedFiles(["libs/protocol/proto/numen/v1/*.proto"])) {
    const { code } = readSource(file);
    for (const hit of findInCode(code, symbols)) found.push({ file, ...hit });
    if (path.basename(file) !== "agent.proto") continue;
    for (const hit of findInCode(code, fields)) found.push({ file, line: hit.line, symbol: `field ${hit.symbol}` });
  }
  return found;
}

/** `numen.v1` holds the contracts of the core and its extension points, and nothing one plugin owns. */
function verifyProtocolHoldsNoPluginContract() {
  checkAgainstBaseline("protocol-holds-no-plugin-contract", "numen.v1 holds what a plugin owns", findPluginProtoSymbols());
}

/** The code that knows no file format and no source kind: it finds a plugin by content type instead. */
const FILE_KIND_FREE_PATHS = [
  "libs/core",
  "libs/notes",
  "libs/plugin-sdk-go",
  "libs/plugin-sdk-ts",
  "libs/ui/core",
  "libs/ui/shell",
  "apps/desktop/editor/src",
];

/** The extensions of the files plugins read: books, documents, recordings and their captions. */
const PLUGIN_EXTENSIONS = [
  "pdf", "epub", "djvu", "fb2", "mobi",
  "mp3", "m4a", "wav", "ogg", "oga", "opus", "flac", "aac", "wma",
  "mp4", "m4v", "mkv", "webm", "mov", "avi",
  "vtt", "srt",
];

/** The source kinds plugins own. */
const PLUGIN_SOURCE_KINDS = ["book", "recording", "url", "document"];

/** A literal that is an extension, a glob of one, or a file name ending in one: `"pdf"`, `".pdf"`, `"onFile:*.pdf"`. */
const EXTENSION_LITERAL = new RegExp(`^(?:[^\\s/]*\\.)?(?:${PLUGIN_EXTENSIONS.join("|")})$`, "i");

/** A literal that is a source kind: `"book"`. */
const SOURCE_KIND_LITERAL = new RegExp(`^(?:${PLUGIN_SOURCE_KINDS.join("|")})$`);

/**
 * The names a source kind takes in code: `KindBook`, `TabRecording`, `SOURCE_KIND_URL`,
 * `SourceKind.BOOK`, `BookFormat`, `DefaultBookExtensions`, `BookProgress`, `ReadBooks`, and the
 * services of the book, document, transcript and article plugins.
 */
const SOURCE_KIND_NAMES = [
  "\\b(\\w*(?:Kind|Tab|Capability)(?:Book|Recording|URL|Url|Document))\\b",
  "\\b(\\w*SOURCE_KIND_(?:BOOK|RECORDING|URL|DOCUMENT)|\\w*BOOK_FORMAT_\\w+)\\b",
  "\\b(\\w*Kinds?\\.(?:BOOK|RECORDING|URL|DOCUMENT))\\b",
  "\\b(\\w*(?:Book|Document)Formats?\\w*)\\b",
  "\\b(\\w*(?:Book|Recording)(?:Extensions|Progress)\\w*)\\b",
  "\\b(\\w*Books\\w*)\\b",
  "\\b(\\w*(?:Book|Document|Transcript|Article)Service\\w*)\\b",
].map((pattern) => new RegExp(pattern, "g"));

/** The constants a script names a source kind by. `URL` is left out: it is also the platform's class. */
const SCRIPT_SOURCE_KIND_NAMES = /(?<![.\w])(BOOK|RECORDING|DOCUMENT)\b(?![\w(])/g;

/** Every literal, and every double-quoted string inside one, such as a JSON manifest a Go raw string holds. */
function getLiteralValues(literals) {
  return literals.flatMap(({ line, value }) => [
    { line, value },
    ...Array.from(value.matchAll(/"((?:[^"\\\n]|\\.)*)"/g), (match) => ({
      line: line + value.slice(0, match.index).split("\n").length - 1,
      value: match[1],
    })),
  ]);
}

/** Every extension literal, source kind literal and source kind name in FILE_KIND_FREE_PATHS. */
function findFileKinds() {
  const found = [];
  for (const file of listShippedFiles(FILE_KIND_FREE_PATHS)) {
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of getLiteralValues(source.literals)) {
      if (EXTENSION_LITERAL.test(value) || SOURCE_KIND_LITERAL.test(value)) {
        found.push({ file, line, symbol: JSON.stringify(value) });
      }
    }
    const patterns = /\.go$/.test(file) ? SOURCE_KIND_NAMES : [...SOURCE_KIND_NAMES, SCRIPT_SOURCE_KIND_NAMES];
    for (const pattern of patterns) {
      for (const hit of findInCode(source.code, pattern)) found.push({ file, ...hit });
    }
  }
  return found;
}

/** The core, the notes library, the SDKs, the UI libraries and the window name no file format and no source kind. */
function verifyNoFileKindOutsidePlugins() {
  checkAgainstBaseline("no-file-kind-outside-plugins", "File formats and source kinds outside plugins", findFileKinds());
}

/** The Markdown editor of `@numen/ui`, which the notes plugin owns, and the wikilink helpers the core replaces. */
const UI_EDITOR = {
  dirs: ["features/editor"],
  modules: ["./features/editor", "./shared/lib/text/address"],
  names: ["Editor", "createTimeGutter", "isNoteAddress", "getWikilinks"],
};

/** Every editor directory `libs/ui/core/src` holds, and every editor or wikilink name `src/index.ts` exports. */
function findUiEditor() {
  const found = UI_EDITOR.dirs
    .filter((dir) => fs.existsSync(path.join(REPO_ROOT, "libs/ui/core/src", dir)))
    .map((dir) => ({ file: `libs/ui/core/src/${dir}`, line: 0, symbol: "directory" }));
  const file = "libs/ui/core/src/index.ts";
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  for (const match of text.matchAll(/^export\s+(?:type\s+)?\{([^}]*)\}\s*from\s*(['"])([^'"]+)\2/gm)) {
    const line = text.slice(0, match.index).split("\n").length;
    const fromEditor = UI_EDITOR.modules.some((one) => match[3] === one || match[3].startsWith(one + "/"));
    for (const name of match[1].split(",").map((one) => one.trim().replace(/^type\s+/, "").split(/\s+as\s+/).pop())) {
      if (name && (fromEditor || UI_EDITOR.names.includes(name))) found.push({ file, line, symbol: `export ${name}` });
    }
  }
  return found;
}

/** `@numen/ui` holds and exports no editor and no wikilink helper. */
function verifyUiExportsNoEditor() {
  checkAgainstBaseline("ui-exports-no-editor", "The UI library holds the editor or wikilink helpers", findUiEditor());
}

/** The ids of the application's windows. */
const WINDOW_IDS = ["editor", "flashcards"];

/** Every window id literal and every name of the window's service in the shipped code of a plugin. */
function findPluginWindowNames() {
  const service = /\b((?:New)?WindowService(?:Client)?)\b/g;
  const found = [];
  for (const file of listShippedFiles(["plugins"])) {
    const source = readSource(file);
    if (!source) continue;
    for (const { line, value } of source.literals) {
      if (WINDOW_IDS.includes(value)) found.push({ file, line, symbol: JSON.stringify(value) });
    }
    for (const hit of findInCode(source.code, service)) found.push({ file, ...hit });
  }
  return found;
}

/** A plugin reaches the window through the SDK: it names no window and creates no `WindowService` client. */
function verifyPluginsNameNoWindow() {
  checkAgainstBaseline("plugins-name-no-window", "A plugin names a window or its service", findPluginWindowNames());
}

/** The calls of Go's `os` package that change the file system. */
const OS_WRITES = /\b(os\.(?:WriteFile|Create|CreateTemp|OpenFile|Mkdir|MkdirAll|MkdirTemp|Rename|Remove|RemoveAll|Symlink|Link|Chmod|Chtimes|Truncate))\(/g;

/** Every `os` write in a shipped Go file of a plugin that names the vault's `.numen` folder. */
function findPluginDataWrites() {
  const found = [];
  for (const file of listShippedFiles(["plugins/*.go"])) {
    const { literals, code } = readSource(file);
    if (!literals.some(({ value }) => value === ".numen" || value.startsWith(".numen/"))) continue;
    for (const hit of findInCode(code, OS_WRITES)) found.push({ file, ...hit });
  }
  return found;
}

/** A plugin writes under `.numen/` through the host store, never with `os`. */
function verifyPluginsWriteDataThroughHost() {
  checkAgainstBaseline("plugins-write-data-through-host", "A plugin writes under .numen/ with os", findPluginDataWrites());
}

const VERIFIERS = [
  {
    id: "core-hexagon",
    scope: "libs",
    module: "core",
    name: "Core Hexagonal Layers & Inward Dependencies (layers_test.go)",
    run: () => {
      exec(
        "go test -v ./container -run \"TestTheLayersAreWhatTheyAre|TestNoAdapterNamesThePortItSatisfies|TestATestOfTheCoreBuildingAnAdapterStandsOutsideIt|TestNothingOfTheCoreReachesTheMachine|TestNoInnerLayerImportsATransport|TestNothingOfTheCoreLogs|TestNothingOfTheCoreReadsTheMachinesClock|TestTheCoresIdentitiesAreItsOwn|TestEveryAdapterServingTheSchemaIsDriving|TestEveryPortIsAskedForSomewhereElse|TestEveryTypePortDeclaresIsNamedSomewhereElse|TestEveryPortMethodIsCalledSomewhereElse|TestNoPackageIsImportedUnderItsLayer|TestTheCoresPublicAdaptersAreTheseAndNoOthers|TestTheCoresHeldAdaptersAreTheseAndNoOthers|TestEveryInstantTheCoreCarriesIsATime|TestTheIdentitiesAPortAsksForAreTheCoresOwn|TestContainerDoesNotDowncastPluginHost\"",
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
    name: "Core Holds No Plugin Package (document, recognition, transcription, links, proofreading, embedding)",
    run: verifyCoreHoldsNoPluginPackage,
  },
  {
    id: "core-imports-no-domain-proto",
    scope: "libs",
    module: "core",
    name: "Core Imports No Domain Proto (transcript, ocr, article, artifact, cards, ...)",
    run: verifyCoreImportsNoDomainProto,
  },
  {
    id: "notes-names-no-producer",
    scope: "libs",
    module: "notes",
    name: "Notes Library Names No Producer (asr, ocr, captions, article, copy, review-log)",
    run: verifyNotesNamesNoProducer,
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
    id: "protocol-holds-no-plugin-contract",
    scope: "libs",
    module: "protocol",
    name: "numen.v1 Holds No Plugin Contract (transcript, article, book, document services, book and recording kinds, card fields)",
    run: verifyProtocolHoldsNoPluginContract,
  },
  {
    id: "no-file-kind-outside-plugins",
    scope: "libs",
    module: "kinds",
    name: "Core, Notes, SDKs, UI and Window Name No File Format or Source Kind (.pdf, .mp3, book, recording, url, document)",
    run: verifyNoFileKindOutsidePlugins,
  },
  {
    id: "ui-isolation",
    scope: "libs",
    module: "ui",
    name: "UI Library Domain & Wire Isolation (0 domain, 0 protocol, 0 wire, 0 shell, 0 apps)",
    run: verifyUiDomainIsolation,
  },
  {
    id: "ui-holds-no-plugin-view",
    scope: "libs",
    module: "ui",
    name: "UI Library Holds No Plugin View (agent chat: Agent, thread, message composer)",
    run: verifyUiHoldsNoPluginView,
  },
  {
    id: "ui-holds-no-shell-part",
    scope: "libs",
    module: "ui",
    name: "UI Library Holds No Window Chrome (workspace, palette, status bar, notices, welcome page)",
    run: verifyUiHoldsNoShellPart,
  },
  {
    id: "ui-exports-no-editor",
    scope: "libs",
    module: "ui",
    name: "UI Library Exports No Editor or Wikilink Helper (features/editor, isNoteAddress, getWikilinks)",
    run: verifyUiExportsNoEditor,
  },
  {
    id: "shell-isolation",
    scope: "libs",
    module: "shell",
    name: "Shell Library Domain & Wire Isolation (0 domain, 0 protocol, 0 wire, 0 apps)",
    run: verifyShellDomainIsolation,
  },
  {
    id: "plugin-host-isolation",
    scope: "libs",
    module: "host",
    name: "Plugin Host Isolation (imports only vue and @numen/plugin-sdk)",
    run: verifyPluginHostIsolation,
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
    name: "Plugins Name No Window (no 'editor' window id, no WindowService client)",
    run: verifyPluginsNameNoWindow,
  },
  {
    id: "plugins-write-data-through-host",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Write Under .numen/ Through the Host Store (no os writes)",
    run: verifyPluginsWriteDataThroughHost,
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
    id: "window-names-no-plugin-kind",
    scope: "apps",
    module: "desktop",
    name: "Desktop Window Names No Plugin Kind (deck, card model, preset, plex, agent, files)",
    run: verifyWindowNamesNoPluginKind,
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

/** The options of a run; every other argument is a target. */
const FLAGS = ["--update-baseline", "--no-baseline", "--list", "--help", "-h"];
const flags = new Set(process.argv.slice(2).filter((arg) => FLAGS.includes(arg)));
const args = process.argv.slice(2).filter((arg) => !FLAGS.includes(arg));

if (flags.has("--help") || flags.has("-h")) {
  console.log("Usage: node check.mjs [target...] [--list] [--help] [--update-baseline] [--no-baseline]");
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
  console.log("");
  console.log("Baseline (baseline.json, read by the checks that list known violations):");
  console.log("  --update-baseline            # remove entries that no longer occur; never adds one");
  console.log("  --no-baseline                # list every violation, known or not");
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
    if (file.startsWith("libs/plugin-sdk-go/")) matched.add("sdk");
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

writeBaseline();

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
