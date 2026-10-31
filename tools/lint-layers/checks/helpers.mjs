import { execSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const CHECKS_DIR = path.dirname(new URL(import.meta.url).pathname);
export const SCRIPT_DIR = path.resolve(CHECKS_DIR, "..");
export const REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");
export const ROOT = REPO_ROOT;
export const MODULES_DIR = REPO_ROOT;

/** The Go module path every module of the repository is named under. */
export const GO_MODULE_ROOT = "github.com/jiva-studio/numen/";

/** The source files read for references, by extension. */
export const SCRIPT_EXTENSIONS = new Set([".ts", ".tsx", ".mts", ".cts", ".js", ".mjs", ".cjs", ".vue"]);

/** The ids and HTTP asset routes of the application's windows. */
export const WINDOW_IDS = ["editor", "flashcards", "desktop.editor", "desktop.flashcards", "mobile"];
export const WINDOW_ROUTE_PREFIXES = ["/assets/"];

/** Tests, stories, test kits of a window, fixtures and generated code. */
export const NOT_SHIPPED = /(_test\.go|\.(test|spec|stories)\.ts|\.pb\.go|\.connect\.go)$|(^|\/)(testing|testdata|e2e|gen)\//;

/** The files that state the boundary between applications and plugins, and so name both sides. */
export const BOUNDARY_RULES = new Set([
  "apps/desktop/editor/eslint.config.js",
  "apps/desktop/internal/layers/linked_test.go",
]);

export function isBoundaryRule(file) {
  return (
    BOUNDARY_RULES.has(file) ||
    /^plugins\/[^/]+\/(eslint\.config\.js|\.golangci(-structure)?\.yml)$/.test(file)
  );
}

export function exec(cmd, cwd = REPO_ROOT) {
  return execSync(cmd, { cwd, stdio: ["ignore", "pipe", "pipe"] });
}

/** Every file under `dir` in the working tree, relative to the repository root. */
export function listFiles(dir) {
  const abs = path.join(REPO_ROOT, dir);
  if (!fs.existsSync(abs)) return [];
  return fs
    .readdirSync(abs, { recursive: true, withFileTypes: true })
    .filter((e) => e.isFile())
    .map((e) => path.relative(REPO_ROOT, path.join(e.parentPath, e.name)));
}

/** The tracked files under `pathspecs` that ship: no test, story, fixture or generated file. */
export function listShippedFiles(pathspecs) {
  const listed = spawnSync("git", ["ls-files", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && !NOT_SHIPPED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
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
export function readScript(source) {
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

/** Lists every string literal of a script with its line, comments left out. */
export function getScriptLiterals(source) {
  return readScript(source).literals;
}

/** The `<script>` blocks of a single-file component, blanked elsewhere so lines keep their numbers. */
export function getVueScript(source) {
  return source.replace(/(^|<\/script>)([\s\S]*?)(<script\b[^>]*>|$)/g, (_, open, between, close) =>
    open + between.replace(/[^\n]/g, " ") + close.replace(/[^\n]/g, " "),
  );
}

/**
 * The module specifiers a TypeScript source names in an `import` or `export … from` statement and
 * in `import()`. A statement starts its line, so the source of a module a string holds is not read.
 */
export function getImportSpecifiers(source) {
  const pattern =
    /^\s*(?:(?:import|export)\s[^'"();=]*?\bfrom|import)\s*(['"])([^'"]+)\1|\bimport\s*\(\s*(['"])([^'"]+)\3/gm;
  return Array.from(source.matchAll(pattern), (match) => match[2] ?? match[4]);
}

/** Every literal, and every double-quoted string inside one, such as a JSON manifest a Go raw string holds. */
export function getLiteralValues(literals) {
  return literals.flatMap(({ line, value }) => [
    { line, value },
    ...Array.from(value.matchAll(/"((?:[^"\\\n]|\\.)*)"/g), (match) => ({
      line: line + value.slice(0, match.index).split("\n").length - 1,
      value: match[1],
    })),
  ]);
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
export function readSource(file) {
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  if (/\.(go|proto)$/.test(file)) return readGo(text);
  if (file.endsWith(".vue")) return readScript(getVueScript(text));
  if (/\.(ts|tsx|mts|cts|js|mjs|cjs)$/.test(file)) return readScript(text);
  return null;
}

/** Every match of `pattern` in the code of a source, with its line and the text it matched. */
export function findInCode(code, pattern) {
  const found = [];
  code.split("\n").forEach((text, index) => {
    for (const match of text.matchAll(pattern)) found.push({ line: index + 1, symbol: match[1] ?? match[0] });
  });
  return found;
}

/** True when `target` is `root` or a path below it. */
export function isWithin(target, root) {
  const relative = path.relative(root, target);
  return relative === "" || (!relative.startsWith("..") && !path.isAbsolute(relative));
}

/**
 * The directories a relative path in `file` is read from: the file's own directory, and every
 * package root above it, which is where a script or a test runs.
 */
export function getPathBases(file) {
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
 * resolves into it, or a match of one of `patterns`. Boundary rule files are not read.
 */
export function findCrossings(from, to, patterns) {
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
    if (isBoundaryRule(file)) return false;
    const text = rest.join(":");
    if (named.some((p) => p.test(text))) return true;
    const bases = getPathBases(file);
    for (const match of text.matchAll(new RegExp(`((?:\\.\\./)+)${to}\\b`, "g"))) {
      if (bases.some((base) => path.resolve(base, match[1] + to) === target)) return true;
    }
    return false;
  });
}

/**
 * Lists every reference a file under `fromDir` makes into `toDir`.
 * A reference is an import path, an alias, a relative file path resolved
 * against the file's directory, or a repository-rooted path.
 */
export function findBoundaryReferences(fromDir, toDir, aliasPattern, packageNames) {
  const toRoot = path.join(REPO_ROOT, toDir);
  const goPrefix = GO_MODULE_ROOT + toDir + "/";
  const listed = spawnSync("git", ["ls-files", "-z", "--", fromDir], { cwd: REPO_ROOT, encoding: "utf8" });
  const files = (listed.stdout || "").split("\0").filter((file) => file && !isBoundaryRule(file));
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
export function getPackageNames(dir) {
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
