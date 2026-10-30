import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { NOT_SHIPPED, REPO_ROOT, failOnFindings, getImportSpecifiers, readGo, requireMatches } from "./helpers.mjs";

const require = createRequire(import.meta.url);

/** A tracked file of the repository read as text. */
function readTrackedSources(pathspecs, accept = () => true) {
  const listed = spawnSync("git", ["ls-files", "-z", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  return (listed.stdout || "")
    .split("\0")
    .filter((file) => file && accept(file) && fs.existsSync(path.join(REPO_ROOT, file)))
    .map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
}

/** True when the first lines of a source mark it as generated. */
export function isGenerated(text) {
  return /^\s*(\/\/|#|\/\*)\s*(Code generated .* DO NOT EDIT|@generated)/m.test(text.split("\n").slice(0, 20).join("\n"));
}

const CODE_FILE = /\.(go|ts|tsx|mts|cts|js|mjs|cjs|vue)$/;

/* ---------------------------- Plugin isolation ---------------------------- */

/** The plugin directory a path belongs to: the capture of `plugins/(*)/`. */
export function getPluginCapture(file) {
  return /^plugins\/([^/]+)\//.exec(file)?.[1];
}

const GO_PLUGIN_IMPORT = /github\.com\/jiva-studio\/numen\/plugins\/([A-Za-z0-9._-]+)/g;
const PATH_TO_PLUGIN = /(?:\.\.\/)+plugins\/([A-Za-z0-9._-]+)|@plugins\/([A-Za-z0-9._-]+)|['"`]plugins\/([A-Za-z0-9._-]+)\//g;

/**
 * Every reference from a file of plugin X to a plugin Y other than X: a Go import, a relative TypeScript or
 * Vue import that resolves into Y, the package name of Y, a path or alias that names Y, and a dependency on
 * the package of Y. `sources` are `{ file, text }` below `plugins/`; `packageNames` maps a plugin directory to
 * its npm package name.
 */
export function findPluginCrossings(sources, packageNames) {
  const plugins = new Set(sources.map(({ file }) => getPluginCapture(file)).filter(Boolean));
  const found = [];
  for (const { file, text } of sources) {
    const own = getPluginCapture(file);
    if (!own) continue;
    const report = (other, how, offset) => {
      if (other && other !== own && plugins.has(other)) {
        found.push({ file, line: text.slice(0, offset).split("\n").length, symbol: `${how} ${other}` });
      }
    };
    if (file.endsWith(".go") || file.endsWith("/go.mod")) {
      for (const match of text.matchAll(GO_PLUGIN_IMPORT)) report(match[1], "Go import of", match.index);
      continue;
    }
    if (path.basename(file) === "package.json") {
      const manifest = JSON.parse(text);
      for (const field of ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"]) {
        for (const name of Object.keys(manifest[field] ?? {})) {
          const other = [...packageNames].find(([, one]) => one === name)?.[0];
          report(other, `${field} names`, 0);
        }
      }
      continue;
    }
    if (!CODE_FILE.test(file) && !/^tsconfig.*\.json$/.test(path.basename(file))) continue;
    for (const match of text.matchAll(PATH_TO_PLUGIN)) report(match[1] ?? match[2] ?? match[3], "path to", match.index);
    for (const specifier of getImportSpecifiers(text)) {
      const offset = Math.max(0, text.indexOf(specifier));
      if (specifier.startsWith(".")) {
        const resolved = path.posix.normalize(path.posix.join(path.posix.dirname(file), specifier));
        report(getPluginCapture(resolved + "/"), "relative import of", offset);
      } else {
        const other = [...packageNames].find(([, name]) => specifier === name || specifier.startsWith(name + "/"))?.[0];
        report(other, "package import of", offset);
      }
    }
  }
  return found;
}

export function verifyPluginIsolation() {
  const sources = readTrackedSources(["plugins"], (file) => !file.includes("/node_modules/"));
  const packageNames = new Map();
  for (const { file, text } of sources) {
    if (/^plugins\/[^/]+\/package\.json$/.test(file)) packageNames.set(getPluginCapture(file), JSON.parse(text).name);
  }
  requireMatches("plugins read", new Set(sources.map(({ file }) => getPluginCapture(file))).size, 2);
  requireMatches("plugin sources read", sources.length, 100);
  failOnFindings("A plugin names another plugin", findPluginCrossings(sources, packageNames));
}

/* ------------------------------ Suppressions ------------------------------ */

const SUPPRESSION_COMMENT = /(?<![\w-])(nolint(?::\S+)?|eslint-disable(?:-next-line|-line)?|eslint-enable|@ts-expect-error|@ts-ignore|@ts-nocheck)(?![\w-])/;
const GOLANGCI_CONFIG = /(^|\/)\.golangci[^/]*\.ya?ml$/;
const GOLANGCI_EXCLUSION_KEY = /^\s*(exclusions|exclude-functions|exclude-rules|exclude-dirs|exclude-files|exclude-use-default)\s*:/;
const ESLINT_CONFIG = /(^|\/)(eslint\.config\.[cm]?js|base-eslint\.mjs)$/;
const SOURCE_FILE_NAME = /^[^*]+\.(ts|tsx|vue|js|mjs|cjs|go)$/;
const REPOSITORY_FILE_PATH = /^(apps|libs|plugins|tools|tests)\/[^*]+\.(ts|tsx|vue|go|mjs|js|json)$/;

/** The comments of a TypeScript or JavaScript source with their lines. */
function getScriptComments(file, text, lineOffset = 0) {
  const ts = require("typescript");
  const kind = file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, kind);
  const seen = new Set();
  const comments = [];
  const take = (ranges) => {
    for (const range of ranges ?? []) {
      if (seen.has(range.pos)) continue;
      seen.add(range.pos);
      comments.push({ line: sourceFile.getLineAndCharacterOfPosition(range.pos).line + 1 + lineOffset, value: text.slice(range.pos, range.end) });
    }
  };
  const visit = (node) => {
    take(ts.getLeadingCommentRanges(text, node.pos));
    take(ts.getTrailingCommentRanges(text, node.end));
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  take(ts.getLeadingCommentRanges(text, sourceFile.endOfFileToken.pos));
  return { comments, sourceFile };
}

/** The comments of a source file, by its language. */
function getComments(file, text) {
  if (file.endsWith(".go")) return readGo(text).comments;
  if (file.endsWith(".vue")) {
    const { parse } = require("@vue/compiler-sfc");
    const { descriptor } = parse(text);
    const scripts = [descriptor.script, descriptor.scriptSetup].filter(Boolean);
    const inScript = scripts.flatMap((block) => getScriptComments(file.replace(/\.vue$/, ".ts"), block.content, block.loc.start.line - 1).comments);
    const inTemplate = [...text.matchAll(/<!--([\s\S]*?)-->/g)].map((match) => ({
      line: text.slice(0, match.index).split("\n").length,
      value: match[0],
    }));
    return [...inScript, ...inTemplate];
  }
  if (CODE_FILE.test(file)) return getScriptComments(file, text).comments;
  return [];
}

/** The string literals of a script that sit in a configuration array of file names: `ignores` and `files`. */
function findFileEntriesInConfig(file, text) {
  const ts = require("typescript");
  const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const found = [];
  const visit = (node) => {
    if (ts.isPropertyAssignment(node) && ts.isArrayLiteralExpression(node.initializer) && ["ignores", "files"].includes(node.name.getText(sourceFile))) {
      for (const element of node.initializer.elements) {
        if (ts.isStringLiteralLike(element) && SOURCE_FILE_NAME.test(path.posix.basename(element.text))) {
          found.push({ file, line: sourceFile.getLineAndCharacterOfPosition(element.getStart(sourceFile)).line + 1, symbol: `file entry ${JSON.stringify(element.text)}` });
        }
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return found;
}

/** The constants of a script that list repository files, which a check would read as exemptions. */
function findFileListsInChecks(file, text) {
  const ts = require("typescript");
  const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const found = [];
  const visit = (node) => {
    if (ts.isVariableDeclaration(node) && node.initializer) {
      let list = node.initializer;
      if (ts.isNewExpression(list) && list.arguments?.length === 1) list = list.arguments[0];
      if (ts.isArrayLiteralExpression(list) && list.elements.length > 0 && list.elements.every((one) => ts.isStringLiteralLike(one) && REPOSITORY_FILE_PATH.test(one.text))) {
        found.push({ file, line: sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1, symbol: `file list ${node.name.getText(sourceFile)}` });
      }
    }
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return found;
}

/**
 * Every suppression in the sources: a lint or type suppression comment, an exclusion in a golangci
 * configuration, a file named in an eslint `ignores` or `files` entry, and a list of repository files in a
 * check. Generated code, found by its header, is not read.
 */
export function findSuppressions(sources) {
  const found = [];
  for (const { file, text } of sources) {
    if (isGenerated(text)) continue;
    for (const { line, value } of getComments(file, text)) {
      const match = SUPPRESSION_COMMENT.exec(value);
      if (match) found.push({ file, line, symbol: match[1] });
    }
    if (GOLANGCI_CONFIG.test(file)) {
      text.split("\n").forEach((one, index) => {
        const key = GOLANGCI_EXCLUSION_KEY.exec(one);
        if (key) found.push({ file, line: index + 1, symbol: key[1] });
        const comment = /#.*\b(nolint)\b/.exec(one);
        if (comment) found.push({ file, line: index + 1, symbol: comment[1] });
      });
    }
    if (ESLINT_CONFIG.test(file)) found.push(...findFileEntriesInConfig(file, text));
    if (/^tools\/lint-layers\/(check\.mjs|checks\/[^/]+\.mjs)$/.test(file)) found.push(...findFileListsInChecks(file, text));
  }
  return found;
}

const SUPPRESSION_FILES = /\.(go|ts|tsx|mts|cts|js|mjs|cjs|vue|ya?ml)$/;

export function verifyNoSuppressions() {
  const sources = readTrackedSources(["."], (file) => SUPPRESSION_FILES.test(file) && !file.includes("node_modules/"));
  requireMatches("sources read for suppressions", sources.length, 500);
  failOnFindings("Suppressions and exemptions", findSuppressions(sources));
}

/* ------------------------------- Duplicates ------------------------------- */

const MIN_DUPLICATE_TOKENS = 40;
const SIMILARITY_THRESHOLD = 0.8;
const SHINGLE = 4;

/** The tokens of a source without comments; whitespace and line breaks do not change them. */
export function getTokens(file, text) {
  const code = file.endsWith(".go") ? readGo(text).code : text.replace(/\/\*[\s\S]*?\*\//g, " ").replace(/(^|[^:'"`])\/\/.*$/gm, "$1");
  return code.match(/[A-Za-z_$][\w$]*|\d+|"(?:[^"\\\n]|\\.)*"|'(?:[^'\\\n]|\\.)*'|`[^`]*`|[^\s\w]/g) ?? [];
}

function getShingles(tokens) {
  const set = new Set();
  for (let i = 0; i + SHINGLE <= tokens.length; i++) set.add(tokens.slice(i, i + SHINGLE).join(" "));
  return set;
}

/** The share of the smaller shingle set that the larger one holds. */
export function getContainment(a, b) {
  const [small, large] = a.size <= b.size ? [a, b] : [b, a];
  if (small.size === 0) return 0;
  let shared = 0;
  for (const one of small) if (large.has(one)) shared++;
  return shared / small.size;
}

/**
 * The files of different plugins that hold the same code: files of one name whose token shingles overlap by
 * the threshold, and files of any name with the same tokens.
 */
export function findDuplicateFiles(sources, { threshold = SIMILARITY_THRESHOLD, languages = /\.(ts|vue|go)$/ } = {}) {
  const files = sources
    .filter(({ file, text }) => getPluginCapture(file) && languages.test(file) && !isGenerated(text))
    .map(({ file, text }) => ({ file, plugin: getPluginCapture(file), tokens: getTokens(file, text) }))
    .filter(({ tokens }) => tokens.length >= MIN_DUPLICATE_TOKENS)
    .map((one) => ({ ...one, shingles: getShingles(one.tokens), hash: one.tokens.join(" ") }));
  const found = [];
  for (let i = 0; i < files.length; i++) {
    for (let j = i + 1; j < files.length; j++) {
      const a = files[i];
      const b = files[j];
      if (a.plugin === b.plugin) continue;
      const isSameName = path.posix.basename(a.file) === path.posix.basename(b.file);
      if (a.hash === b.hash || (isSameName && getContainment(a.shingles, b.shingles) >= threshold)) {
        found.push({ file: a.file, line: 1, symbol: `duplicates ${b.file}` });
      }
    }
  }
  return found;
}

/** Configuration files that every plugin repeats by design, kept equal by the golden-parity check. */
const PLUGIN_CONFIG_FILE = /(^|\/)(vite|vitest|eslint|stories)\.config\.[cm]?[jt]s$/;

function readShippedPluginSources() {
  const sources = readTrackedSources(["plugins"], (file) => !file.includes("/node_modules/") && !NOT_SHIPPED.test(file) && !PLUGIN_CONFIG_FILE.test(file));
  requireMatches("plugin sources read", sources.length, 100);
  return sources;
}

export function verifyNoDuplicateFrontendFiles() {
  failOnFindings("Frontend files of different plugins hold the same code", findDuplicateFiles(readShippedPluginSources(), { languages: /\.(ts|vue)$/ }));
}

export function verifyNoDuplicateBackendFiles() {
  failOnFindings("Go files of different plugins hold the same code", findDuplicateFiles(readShippedPluginSources(), { languages: /\.go$/ }));
}

/* ----------------------------- Import cycles ------------------------------ */

const IMPORT_EXTENSIONS = ["", ".ts", ".tsx", ".vue", ".js", ".mjs", "/index.ts", "/index.vue", "/index.js"];

/** The nearest directory at or above `file` that holds a package.json, from a set of known directories. */
function getPackageRoot(file, packageRoots) {
  for (let dir = path.posix.dirname(file); dir !== "."; dir = path.posix.dirname(dir)) if (packageRoots.has(dir)) return dir;
  return "";
}

/** The file a relative or `@/` import of `file` names among `known`, or undefined. */
function resolveImport(file, specifier, known, packageRoots) {
  let base;
  if (specifier.startsWith(".")) base = path.posix.normalize(path.posix.join(path.posix.dirname(file), specifier));
  else if (specifier.startsWith("@/")) base = path.posix.join(getPackageRoot(file, packageRoots), "src", specifier.slice(2));
  else return undefined;
  return IMPORT_EXTENSIONS.map((extension) => base + extension).find((candidate) => known.has(candidate));
}

/** The groups of files that import each other in a cycle, by strongly connected components. */
export function findImportCycles(sources, packageRoots) {
  const known = new Set(sources.map(({ file }) => file));
  const edges = new Map();
  for (const { file, text } of sources) {
    const targets = new Set();
    const runtimeText = text.replace(/^\s*(?:import|export)\s+type\b[\s\S]*?\bfrom\s*(['"])[^'"]+\1/gm, "");
    for (const specifier of getImportSpecifiers(runtimeText)) {
      const target = resolveImport(file, specifier, known, packageRoots);
      if (target) targets.add(target);
    }
    edges.set(file, [...targets]);
  }
  let counter = 0;
  const index = new Map();
  const low = new Map();
  const onStack = new Set();
  const stack = [];
  const groups = [];
  const connect = (node) => {
    index.set(node, counter);
    low.set(node, counter);
    counter++;
    stack.push(node);
    onStack.add(node);
    for (const next of edges.get(node) ?? []) {
      if (!index.has(next)) {
        connect(next);
        low.set(node, Math.min(low.get(node), low.get(next)));
      } else if (onStack.has(next)) {
        low.set(node, Math.min(low.get(node), index.get(next)));
      }
    }
    if (low.get(node) === index.get(node)) {
      const group = [];
      let member;
      do {
        member = stack.pop();
        onStack.delete(member);
        group.push(member);
      } while (member !== node);
      if (group.length > 1 || (edges.get(node) ?? []).includes(node)) groups.push(group.sort());
    }
  };
  for (const file of known) if (!index.has(file)) connect(file);
  return groups.map((group) => ({ file: group[0], line: 1, symbol: `import cycle of ${group.length}: ${group.join(" -> ")}` }));
}

const FRONTEND_SHIPPED_SOURCE = /^(apps\/.+\/src|libs\/.+\/src|plugins\/[^/]+\/frontend)\/.*\.(ts|vue)$/;

export function verifyNoImportCycles() {
  const listed = spawnSync("git", ["ls-files", "-z"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const all = (listed.stdout || "").split("\0").filter(Boolean);
  const packageRoots = new Set(all.filter((file) => file.endsWith("/package.json")).map((file) => path.posix.dirname(file)));
  const files = all.filter((file) => FRONTEND_SHIPPED_SOURCE.test(file) && !NOT_SHIPPED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
  requireMatches("frontend sources read for cycles", files.length, 500);
  const sources = files.map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
  failOnFindings("Frontend files import each other in a cycle at runtime", findImportCycles(sources, packageRoots));
}

/* ------------------------------- Assignment ------------------------------- */

/** The top-level directories that hold layers of the repository, and the directories that hold none. */
export const LAYER_ROOTS = ["libs", "plugins", "apps"];
export const AUXILIARY_ROOTS = ["tools", "tests", "docs"];

/**
 * The modules and files that no layer owns: a Go module or npm package outside the layer roots and the
 * auxiliary roots, a file directly in a layer root, and a plugin directory without a manifest.
 */
export function findUnassignedPaths(files) {
  const found = [];
  const known = new Set([...LAYER_ROOTS, ...AUXILIARY_ROOTS]);
  const manifestDirs = new Set(files.filter((file) => /^plugins\/[^/]+\/manifest\.json$/.test(file)).map((file) => file.split("/")[1]));
  for (const file of files) {
    const parts = file.split("/");
    const isModule = ["go.mod", "package.json"].includes(parts[parts.length - 1]);
    if (isModule && parts.length > 1 && !known.has(parts[0])) found.push({ file, line: 1, symbol: `module in the unassigned directory ${parts[0]}/` });
    if (LAYER_ROOTS.includes(parts[0]) && parts.length === 2 && !parts[1].startsWith(".")) {
      found.push({ file, line: 1, symbol: `file directly in ${parts[0]}/` });
    }
    if (parts[0] === "plugins" && parts.length > 2 && !parts[1].startsWith(".") && !manifestDirs.has(parts[1])) {
      found.push({ file, line: 1, symbol: `plugin ${parts[1]} has no manifest.json` });
    }
  }
  return found;
}

export function verifyEveryPathHasALayer() {
  const listed = spawnSync("git", ["ls-files", "-z"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const files = (listed.stdout || "").split("\0").filter(Boolean);
  requireMatches("tracked files read", files.length, 1000);
  requireMatches("plugin manifests read", files.filter((file) => /^plugins\/[^/]+\/manifest\.json$/.test(file)).length, 2);
  failOnFindings("Paths that no layer owns", findUnassignedPaths(files));
}

/* --------------------------------- go.work -------------------------------- */

/** The module directories a go.work file uses. */
export function parseWorkspaceUses(text) {
  const dirs = [];
  let isBlock = false;
  for (const raw of text.split("\n")) {
    const line = raw.replace(/\/\/.*$/, "").trim();
    if (/^use\s*\($/.test(line)) isBlock = true;
    else if (isBlock && line === ")") isBlock = false;
    else if (isBlock && line) dirs.push(line.replace(/^\.\//, ""));
    else if (/^use\s+\S+/.test(line)) dirs.push(line.replace(/^use\s+/, "").replace(/^\.\//, ""));
  }
  return dirs;
}

/** The disagreement between a go.work file and the directories that hold a go.mod. */
export function findWorkspaceGaps(workText, moduleDirs) {
  const used = new Set(parseWorkspaceUses(workText));
  const found = [];
  for (const dir of moduleDirs) if (!used.has(dir)) found.push({ file: "go.work", line: 1, symbol: `missing use ./${dir}` });
  for (const dir of used) if (!moduleDirs.includes(dir)) found.push({ file: "go.work", line: 1, symbol: `use ./${dir} has no go.mod` });
  return found;
}

export function verifyWorkspaceListsEveryModule() {
  const listed = spawnSync("git", ["ls-files", "-z", "--", "go.mod", "**/go.mod"], { cwd: REPO_ROOT, encoding: "utf8" });
  const moduleDirs = (listed.stdout || "")
    .split("\0")
    .filter(Boolean)
    .map((file) => path.posix.dirname(file))
    .filter((dir) => dir !== ".");
  requireMatches("go.mod files read", moduleDirs.length, 10);
  failOnFindings("go.work disagrees with the modules", findWorkspaceGaps(fs.readFileSync(path.join(REPO_ROOT, "go.work"), "utf8"), moduleDirs));
}

export const DISCIPLINE_VERIFIERS = [
  {
    id: "plugins-isolation-capture",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Isolation (one rule over plugins/(*)/: Go and TypeScript references name no other plugin)",
    run: verifyPluginIsolation,
  },
  {
    id: "no-suppressions",
    scope: "libs",
    module: "quality",
    name: "Zero Suppressions (no nolint, eslint-disable, @ts-ignore, @ts-expect-error, exclusions, file exemptions)",
    run: verifyNoSuppressions,
  },
  {
    id: "plugins-share-no-frontend-file",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Share No Frontend File (no TypeScript or Vue file of one plugin repeats a file of another)",
    run: verifyNoDuplicateFrontendFiles,
  },
  {
    id: "plugins-share-no-backend-file",
    scope: "plugins",
    module: "plugins",
    isManual: true,
    name: "Plugins Share No Backend File (no Go file of one plugin repeats a file of another)",
    run: verifyNoDuplicateBackendFiles,
  },
  {
    id: "frontend-no-import-cycle",
    scope: "libs",
    module: "quality",
    name: "Frontend Files Form No Import Cycle (apps, libs and plugin frontends)",
    run: verifyNoImportCycles,
  },
  {
    id: "every-path-has-a-layer",
    scope: "libs",
    module: "quality",
    name: "Every Module Belongs To A Layer (libs, plugins, apps; fail-closed)",
    run: verifyEveryPathHasALayer,
  },
  {
    id: "workspace-lists-every-module",
    scope: "libs",
    module: "quality",
    name: "go.work Lists Every Module That Has a go.mod",
    run: verifyWorkspaceListsEveryModule,
  },
];
