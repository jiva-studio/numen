import { execSync } from "node:child_process";
import { runTool } from "../../shared/tools.mjs";
import { parse as parseVue } from "@vue/compiler-sfc";
import fs from "node:fs";
import os from "node:os";
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

/** Every tracked file of the repository that exists on disk. */
export function listTrackedFiles() {
  const listed = runTool("git", ["ls-files"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 1 << 28 });
  return (listed.stdout || "").split("\n").filter((file) => file && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** The tracked files under `pathspecs` that ship: no test, story, fixture or generated file. */
export function listShippedFiles(pathspecs) {
  const listed = runTool("git", ["ls-files", "--", ...pathspecs], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && !NOT_SHIPPED.test(file) && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** Fails when the selector of a rule matched nothing, so a rule over an empty set cannot pass. */
export function requireMatches(title, count) {
  if (count < 1) {
    throw new Error(`${title}: the selector matched nothing; the rule examined nothing`);
  }
}

/** Fails with every finding, by file, line and symbol. */
export function failOnFindings(title, findings) {
  if (findings.length === 0) return;
  const listed = findings.map(({ file, line, symbol }) => `${file}:${line}: ${symbol}`);
  throw new Error(`${title} (${findings.length}):\n` + listed.join("\n"));
}

/**
 * Lists every string literal of a script with its line, read from its TypeScript syntax tree. A
 * template literal is read up to its first substitution.
 */
function readScriptLiterals(file, source, lineOffset = 0) {
  const kind = file.endsWith(".tsx") ? ts.ScriptKind.TSX : ts.ScriptKind.TS;
  const sourceFile = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true, kind);
  const literals = [];
  const visit = (node) => {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      const line = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile)).line + 1 + lineOffset;
      literals.push({ line, value: node.text });
    } else if (ts.isTemplateExpression(node)) {
      const line = sourceFile.getLineAndCharacterOfPosition(node.head.getStart(sourceFile)).line + 1 + lineOffset;
      literals.push({ line, value: node.head.text });
    }
    ts.forEachChild(node, visit);
  };
  visit(sourceFile);
  return { literals };
}

/** Runs `go <subcommand> edit -json` on `source` written to a temporary file, and returns the parsed JSON. */
function runGoFileJson(subcommand, source) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), `numen-go-${subcommand}-`));
  const file = path.join(dir, `go.${subcommand}`);
  try {
    fs.writeFileSync(file, source, "utf8");
    const run = runTool("go", [subcommand, "edit", "-json", file], { cwd: dir, encoding: "utf8" });
    if (run.status !== 0) {
      throw new Error(`go ${subcommand} edit -json failed: ${(run.stderr || run.stdout || "").trim()}`);
    }
    return JSON.parse(run.stdout);
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

/** The parsed JSON structure of a `go.mod` file, read via `go mod edit -json`. */
export function parseGoMod(source) {
  return runGoFileJson("mod", source);
}

/** The parsed JSON structure of a `go.work` file, read via `go work edit -json`. */
export function parseGoWork(source) {
  return runGoFileJson("work", source);
}

/** Lists the import paths of a Go file with their lines, and its other string literals. */
export function getGoReferences(source) {
  const { imports, literals } = readGo(source);
  return { imports, literals };
}

/** Lists the module paths and local replacement paths a `go.mod` names, read via `go mod edit -json`. */
export function getGoModReferences(source) {
  const parsed = parseGoMod(source);
  const codeLines = readGo(source).code.split("\n");
  const findLine = (token, fromIndex = 0) => {
    for (let i = fromIndex; i < codeLines.length; i++) {
      if (codeLines[i].includes(token)) return i + 1;
    }
    return 1;
  };
  const imports = [];
  const literals = [];
  for (const req of parsed.Require ?? []) {
    if (req.Path?.startsWith(GO_MODULE_ROOT)) {
      imports.push({ line: findLine(req.Path), value: req.Path });
    }
  }
  for (const rep of parsed.Replace ?? []) {
    const line = findLine(rep.Old?.Path ?? "=>");
    for (const candidate of [rep.Old?.Path, rep.New?.Path]) {
      if (candidate?.startsWith(GO_MODULE_ROOT)) {
        imports.push({ line, value: candidate });
      }
    }
    if (rep.New?.Path && (rep.New.Path.startsWith("./") || rep.New.Path.startsWith("../"))) {
      literals.push({ line, value: rep.New.Path });
    }
  }
  return { imports, literals };
}

/**
 * Reads the Go string literal whose opening quote is at `start`: its value with escapes resolved to
 * the escaped character, and the index past its closing quote.
 */
function readGoLiteral(source, start) {
  const quote = source[start];
  let value = "";
  const isRaw = quote === "`";
  let i = start + 1;
  while (i < source.length) {
    const c = source[i];
    if (c === quote || (!isRaw && c === "\n")) break;
    if (!isRaw && c === "\\") {
      value += source[i + 1] ?? "";
      i += 2;
    } else {
      value += c;
      i++;
    }
  }
  return { value, end: i + 1 };
}

/**
 * Lists the import paths, other string literals and comments of a Go or proto source with their lines,
 * and returns the source with its comments and literals blanked so lines keep their numbers.
 */
export function readGo(source) {
  const imports = [];
  const literals = [];
  const comments = [];
  let code = "";
  let line = 1;
  let i = 0;
  let importState = "none";
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
      const startLine = line;
      const literal = readGoLiteral(source, i);
      i = literal.end;
      if (importState === "afterKeyword") {
        imports.push({ line: startLine, value: literal.value });
        importState = "none";
      } else if (importState === "inBlock") {
        imports.push({ line: startLine, value: literal.value });
      } else {
        literals.push({ line: startLine, value: literal.value });
      }
      const text = source.slice(from, i);
      line += text.split("\n").length - 1;
      code += blank(text);
      continue;
    }
    if (/[A-Za-z_]/.test(c)) {
      while (i < source.length && /[A-Za-z0-9_]/.test(source[i])) i++;
      const word = source.slice(from, i);
      if (word === "import" && importState === "none") importState = "afterKeyword";
      code += word;
      continue;
    }
    if (c === "(" && importState === "afterKeyword") importState = "inBlock";
    else if (c === ")" && importState === "inBlock") importState = "none";
    else if ((c === "\n" || c === ";") && importState === "afterKeyword") importState = "none";
    if (c === "\n") line++;
    code += c;
    i++;
  }
  return { imports, literals, comments, code };
}

/** The parsed `package.json` text of a package. */
export function parsePackageJson(text) {
  return JSON.parse(text);
}

/** Reads a tracked source file as literals and code, by its extension; null for any other file. */
export function readSource(file) {
  const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
  if (/\.(go|proto)$/.test(file)) {
    const { imports, literals, comments, code } = readGo(text);
    return { literals: [...imports, ...literals], comments, code };
  }
  if (file.endsWith(".vue")) {
    const { descriptor } = parseVue(text);
    const scripts = [descriptor.script, descriptor.scriptSetup].filter(Boolean);
    const literals = scripts.flatMap(
      (block) => readScriptLiterals(file.replace(/\.vue$/, ".ts"), block.content, block.loc.start.line - 1).literals,
    );
    return { literals };
  }
  if (/\.(ts|tsx|mts|cts|js|mjs|cjs)$/.test(file)) return readScriptLiterals(file, text);
  return null;
}

/** The module specifiers of the import and export declarations of a script, read from its syntax tree. */
export function getModuleSpecifiers(file, text) {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  return source.statements
    .filter((node) => (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) && node.moduleSpecifier && ts.isStringLiteral(node.moduleSpecifier))
    .map((node) => node.moduleSpecifier.text);
}
