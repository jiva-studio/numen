import { execSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";

const CHECKS_DIR = path.dirname(new URL(import.meta.url).pathname);
export const SCRIPT_DIR = path.resolve(CHECKS_DIR, "..");
export const REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");
export const ROOT = REPO_ROOT;
export const MODULES_DIR = REPO_ROOT;

/** The Go module path every module of the repository is named under. */
export const GO_MODULE_ROOT = "github.com/jiva-studio/numen/";

/** Tests, stories, test kits of a window, fixtures and generated code. */
const NOT_SHIPPED = /(_test\.go|\.(test|spec|stories)\.ts|\.pb\.go|\.connect\.go)$|(^|\/)(testing|testdata|fixtures|e2e|gen)\//;

export function exec(cmd, cwd = REPO_ROOT) {
  return execSync(cmd, { cwd, stdio: ["ignore", "pipe", "pipe"] });
}

/** The tracked files under `pathspecs` that ship: no test, story, fixture or generated file. */
export function listShippedFiles(pathspecs) {
  const listed = spawnSync("git", ["ls-files", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && !NOT_SHIPPED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** Fails when fewer than `minimum` things were examined, so a rule over an empty set cannot pass. */
export function requireMatches(title, count, minimum) {
  if (count < minimum) {
    throw new Error(`${title}: ${count} found, at least ${minimum} expected; the rule examined too little`);
  }
}

/** Fails with every finding, by file, line and symbol. */
export function failOnFindings(title, findings) {
  if (findings.length === 0) return;
  const listed = findings.map(({ file, line, symbol }) => `${file}:${line}: ${symbol}`);
  throw new Error(`${title} (${findings.length}):\n` + listed.join("\n"));
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
export function getGoReferences(source) {
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
export function getGoModReferences(source) {
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

/**
 * Lists every string literal of a Go or proto source with its line, and returns the source with its
 * comments and literals blanked so lines keep their numbers.
 */
export function readGo(source) {
  const literals = [];
  const comments = [];
  let code = "";
  let line = 1;
  let i = 0;
  const blank = (text) => text.replace(/[^\n]/g, " ");
  while (i < source.length) {
    const c = source[i];
    const from = i;
    if (c === "/" && source[i + 1] === "/") {
      while (i < source.length && source[i] !== "\n") i++;
      comments.push({ line, value: source.slice(from, i) });
      code += blank(source.slice(from, i));
      continue;
    }
    if (c === "/" && source[i + 1] === "*") {
      const end = source.indexOf("*/", i + 2);
      i = end < 0 ? source.length : end + 2;
      const text = source.slice(from, i);
      comments.push({ line, value: text });
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
  return { literals, comments, code };
}

/** The parsed `package.json` text of a package. */
export function parsePackageJson(text) {
  return JSON.parse(text);
}

/** Reads a tracked source file as literals and code, by its extension; null for any other file. */
export function readSource(file) {
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  if (/\.(go|proto)$/.test(file)) return readGo(text);
  if (file.endsWith(".vue")) return readScript(getVueScript(text));
  if (/\.(ts|tsx|mts|cts|js|mjs|cjs)$/.test(file)) return readScript(text);
  return null;
}

/** The module specifiers of the import and export declarations of a script, read from its syntax tree. */
export function getModuleSpecifiers(file, text) {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  return source.statements
    .filter((node) => (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier))
    .map((node) => node.moduleSpecifier.text);
}
