import { runTool } from "../../shared/tools.mjs";
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { REPO_ROOT, failOnFindings, readGo, removeLineComment, requireMatches } from "./helpers.mjs";

const require = createRequire(import.meta.url);

/** A tracked file of the repository read as text. */
function readTrackedSources(pathspecs, accept = () => true) {
  const listed = runTool("git", ["ls-files", "-z", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  return (listed.stdout || "")
    .split("\0")
    .filter((file) => file && accept(file) && fs.existsSync(path.join(REPO_ROOT, file)))
    .map((file) => ({ file, text: fs.readFileSync(path.join(REPO_ROOT, file), "utf8") }));
}

/** True when the first lines of a source mark it as generated. */
export function isGenerated(text) {
  return /^[ \t]*(\/\/|#|\/\*)\s*(Code generated .* DO NOT EDIT|@generated)/m.test(text.split("\n").slice(0, 20).join("\n"));
}

const CODE_FILE = /\.(go|ts|tsx|mts|cts|js|mjs|cjs|vue)$/;

/* ---------------------------- Plugin isolation ---------------------------- */

/** The plugin directory a path belongs to: the capture of `plugins/(*)/`. */
export function getPluginCapture(file) {
  return /^plugins\/([^/]+)\//.exec(file)?.[1];
}

const GO_PLUGIN_IMPORT = /github\.com\/jiva-studio\/numen\/plugins\/([A-Za-z0-9._-]+)/g;

/**
 * Every reference from a file of plugin X to a plugin Y other than X: a Go import, and a dependency on the
 * package of Y; the imports of scripts are checked by `dependency-graph`. `sources` are `{ file, text }`
 * below `plugins/`; `packageNames` maps a plugin directory to its npm package name.
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

/** The schema files of plugins: a plugin reaches the core through the SDK and declares no schema of its own. */
export function findPluginSchemas(files) {
  return files.filter((file) => getPluginCapture(file) && file.endsWith(".proto")).map((file) => ({ file, line: 1, symbol: "a plugin holds a protobuf schema" }));
}

export function verifyPluginsHoldNoSchema() {
  const listed = runTool("git", ["ls-files", "-z", "--", "plugins"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  const files = (listed.stdout || "").split("\0").filter(Boolean);
  requireMatches("plugin files read", files.length, 100);
  failOnFindings("A plugin declares its own schema", findPluginSchemas(files));
}

/* ------------------------------ Suppressions ------------------------------ */

const SUPPRESSION_COMMENT = /(?<![\w-])(nolint(?::\S+)?|eslint-disable(?:-next-line|-line)?|eslint-enable|@ts-expect-error|@ts-ignore|@ts-nocheck)(?![\w-])/;
const GOLANGCI_CONFIG = /(^|\/)\.golangci[^/]*\.ya?ml$/;
const GOLANGCI_EXCLUSION_KEY = /^\s*(exclusions|exclude-functions|exclude-rules|exclude-dirs|exclude-files|exclude-use-default)\s*:/;
const ESLINT_CONFIG = /(^|\/)(eslint\.config\.[cm]?js|base-eslint\.mjs)$/;
const SOURCE_FILE_NAME = /^[^*]+\.(ts|tsx|vue|js|mjs|cjs|go)$/;
/** The scripts that state rules: the checks and the configurations of the tools they run. */
const CHECK_SCRIPT = /^(tools\/lint-layers\/.+\.mjs|stylelint\.config\.mjs)$/;
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

/** The arrays of a check that list repository files, which it would read as exemptions: in a constant or inline. */
export function findFileListsInChecks(file, text) {
  const ts = require("typescript");
  const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const found = [];
  const visit = (node) => {
    if (ts.isArrayLiteralExpression(node) && node.elements.length > 0 && node.elements.every((one) => ts.isStringLiteralLike(one) && REPOSITORY_FILE_PATH.test(one.text))) {
      const owner = ts.isNewExpression(node.parent) ? node.parent.parent : node.parent;
      const name = ts.isVariableDeclaration(owner) ? owner.name.getText(sourceFile) : "inline";
      found.push({ file, line: sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1, symbol: `file list ${name}` });
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
        const hash = one.indexOf("#");
        if (hash !== -1 && /\bnolint\b/.test(one.slice(hash))) found.push({ file, line: index + 1, symbol: "nolint" });
      });
    }
    if (ESLINT_CONFIG.test(file)) found.push(...findFileEntriesInConfig(file, text));
    if (CHECK_SCRIPT.test(file) && !file.endsWith(".test.mjs")) found.push(...findFileListsInChecks(file, text));
  }
  return found;
}

const SUPPRESSION_FILES = /\.(go|ts|tsx|mts|cts|js|mjs|cjs|vue|ya?ml)$/;

export function verifyNoSuppressions() {
  const sources = readTrackedSources(["."], (file) => SUPPRESSION_FILES.test(file) && !file.includes("node_modules/"));
  requireMatches("sources read for suppressions", sources.length, 500);
  failOnFindings("Suppressions and exemptions", findSuppressions(sources));
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
  const listed = runTool("git", ["ls-files", "-z"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
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
    const line = removeLineComment(raw).trim();
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
  const listed = runTool("git", ["ls-files", "-z", "--", "go.mod", "**/go.mod"], { cwd: REPO_ROOT, encoding: "utf8" });
  const moduleDirs = (listed.stdout || "")
    .split("\0")
    .filter(Boolean)
    .map((file) => path.posix.dirname(file))
    .filter((dir) => dir !== ".");
  requireMatches("go.mod files read", moduleDirs.length, 10);
  failOnFindings("go.work disagrees with the modules", findWorkspaceGaps(fs.readFileSync(path.join(REPO_ROOT, "go.work"), "utf8"), moduleDirs));
}

/* -------------------------------- JSON keys ------------------------------- */

function isCamelCaseKey(name) {
  return /^[a-z][a-zA-Z0-9]*$/.test(name);
}

function isSnakeCaseKey(name) {
  return /^[A-Za-z0-9]+_[A-Za-z0-9_]+$/.test(name);
}

function isRpcMethodSignature(node, ts) {
  if (!node.type || !ts.isTypeLiteralNode(node.type)) return false;
  const names = node.type.members
    .filter((member) => ts.isPropertySignature(member) && member.name && ts.isIdentifier(member.name))
    .map((member) => member.name.text);
  return names.includes("request") && names.includes("response");
}

function findJsonObjectKeyViolations(file, text) {
  const found = [];
  const visit = (value) => {
    if (Array.isArray(value)) {
      for (const item of value) visit(item);
      return;
    }
    if (!value || typeof value !== "object") return;
    for (const [key, child] of Object.entries(value)) {
      if (!isCamelCaseKey(key)) found.push({ file, line: 1, symbol: `JSON key ${JSON.stringify(key)}` });
      visit(child);
    }
  };
  visit(JSON.parse(text));
  return found;
}

function findScriptJsonKeyViolations(file, text, lineOffset = 0) {
  const ts = require("typescript");
  const kind = file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, kind);
  const isApiFile = /(^|\/)api(\/|\.ts$)/.test(file);
  const found = [];
  const checkProperty = (node) => {
    const name = node.name && (ts.isIdentifier(node.name) || ts.isStringLiteral(node.name)) ? node.name.text : "";
    if (!name) return;
    const isInvalid = isSnakeCaseKey(name) || (isApiFile && !isRpcMethodSignature(node, ts) && !isCamelCaseKey(name));
    if (!isInvalid) return;
    const line = sourceFile.getLineAndCharacterOfPosition(node.name.getStart(sourceFile)).line + 1 + lineOffset;
    found.push({ file, line, symbol: `JSON property ${JSON.stringify(name)}` });
  };
  const visit = (node) => {
    if (ts.isPropertySignature(node)) checkProperty(node);
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return found;
}

/**
 * Every non-camelCase JSON key in project manifests, API fixtures, and TypeScript JSON shapes.
 */
export function findNonCamelCaseJsonKeys(sources) {
  const found = [];
  for (const { file, text } of sources) {
    if (isGenerated(text)) continue;
    if (file.endsWith(".json")) {
      found.push(...findJsonObjectKeyViolations(file, text));
    } else if (file.endsWith(".vue")) {
      const { parse } = require("@vue/compiler-sfc");
      const { descriptor } = parse(text);
      for (const block of [descriptor.script, descriptor.scriptSetup].filter(Boolean)) {
        found.push(...findScriptJsonKeyViolations(file.replace(/\.vue$/, ".ts"), block.content, block.loc.start.line - 1));
      }
    } else if (/\.(ts|tsx)$/.test(file)) {
      found.push(...findScriptJsonKeyViolations(file, text));
    }
  }
  return found;
}

const PROJECT_JSON_OR_TS = /(\/manifest\.json|\/testdata\/api\/[^/]+\.json|\.(ts|tsx|vue))$/;

export function verifyJsonKeysAreCamelCase() {
  const sources = readTrackedSources(LAYER_ROOTS, (file) => PROJECT_JSON_OR_TS.test(file) && !file.includes("node_modules/"));
  requireMatches("manifests and TypeScript sources read for JSON keys", sources.length, 500);
  failOnFindings("Non-camelCase JSON keys", findNonCamelCaseJsonKeys(sources));
}

export const DISCIPLINE_VERIFIERS = [
  {
    id: "plugins-isolation-capture",
    scope: "plugins",
    module: "plugins",
    name: "Plugin Isolation (one rule over plugins/(*)/: Go imports and package dependencies name no other plugin)",
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
    id: "plugins-hold-no-schema",
    scope: "plugins",
    module: "plugins",
    name: "Plugins Hold No Schema (no .proto file under plugins/(*)/)",
    run: verifyPluginsHoldNoSchema,
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
  {
    id: "json-keys-camelcase",
    scope: "libs",
    module: "quality",
    name: "JSON Keys Are camelCase (manifest schemas, API fixtures, and TypeScript JSON types)",
    run: verifyJsonKeysAreCamelCase,
  },
];
