import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { REPO_ROOT, failOnFindings, listTrackedFiles, requireMatches } from "../../../lib/graph.mjs";

/** The extensions a relative specifier may leave out, in the order a bundler tries them. */
const EXTENSIONS = [".ts", ".mts", ".js", ".mjs", ".vue"];

/** True when a path has a `testing` folder. */
const isTestingModule = (file) => file.split("/").includes("testing");

/** The relative specifiers of the imports, exports and `import()` calls of a script. */
function readRelativeSpecifiers(file, text) {
  const source = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
  const found = [];
  const add = (literal) => {
    if (literal && ts.isStringLiteralLike(literal) && literal.text.startsWith(".")) found.push(literal.text);
  };
  const visit = (node) => {
    if (ts.isImportDeclaration(node) || ts.isExportDeclaration(node)) add(node.moduleSpecifier);
    else if (ts.isCallExpression(node) && node.expression.kind === ts.SyntaxKind.ImportKeyword) add(node.arguments[0]);
    ts.forEachChild(node, visit);
  };
  visit(source);
  return found;
}

/** The file a relative specifier of `from` names among `files`, or undefined when it names none of them. */
function resolveSpecifier(from, specifier, files) {
  const base = path.posix.normalize(path.posix.join(path.posix.dirname(from), specifier));
  return [base, ...EXTENSIONS.map((one) => base + one), ...EXTENSIONS.map((one) => `${base}/index${one}`)].find((one) => files.has(one));
}

/** The first target of an `exports` entry: a string, or the first string among the conditions of an object. */
function readTarget(target) {
  if (typeof target === "string") return target;
  if (!target || typeof target !== "object") return undefined;
  return Object.values(target).map(readTarget).find(Boolean);
}

/** The entries of a package as `{ entry, file }` with `file` relative to the root: its `exports`, else its `types` or `main`. */
function readEntries(packageDir, manifest, files) {
  const exported = typeof manifest.exports === "string" ? { ".": manifest.exports } : manifest.exports;
  const targets = exported ? Object.entries(exported) : [[".", manifest.types ?? manifest.main]];
  return targets.flatMap(([entry, target]) => {
    const one = readTarget(target);
    if (!one || one.includes("*")) return [];
    const file = path.posix.normalize(path.posix.join(packageDir, one));
    return files.has(file) ? [{ entry, file }] : [];
  });
}

/**
 * Lists the `testing` modules a non-testing entry of a package reaches through its relative imports.
 * `files` are the tracked paths from the root, `readText` reads one. Returns the findings and the
 * number of entries read.
 */
export function findTestingInEntries(files, readText) {
  const all = new Set(files);
  const findings = [];
  let entries = 0;
  for (const manifestFile of files.filter((one) => one.startsWith("libs/") && one.endsWith("/package.json") && !one.includes("node_modules/"))) {
    const packageDir = path.posix.dirname(manifestFile);
    for (const { entry, file } of readEntries(packageDir, JSON.parse(readText(manifestFile)), all)) {
      if (isTestingModule(file)) continue;
      entries++;
      const parent = new Map([[file, undefined]]);
      const queue = [file];
      while (queue.length > 0) {
        const current = queue.shift();
        if (isTestingModule(current)) {
          const chain = [];
          for (let one = current; one; one = parent.get(one)) chain.unshift(one);
          findings.push({ file: current, line: 1, symbol: `the entry ${entry} of ${packageDir} reaches it through ${chain.join(" > ")}` });
          continue;
        }
        if (!/\.(ts|mts|js|mjs)$/.test(current)) continue;
        for (const specifier of readRelativeSpecifiers(current, readText(current))) {
          const next = resolveSpecifier(current, specifier, all);
          if (next && !parent.has(next)) {
            parent.set(next, current);
            queue.push(next);
          }
        }
      }
    }
  }
  return { findings, entries };
}

/** No entry of a library other than its testing entries reaches a `testing` module, so test code never ships in a bundle. */
export function verifyNoTestingInEntries() {
  const { findings, entries } = findTestingInEntries(listTrackedFiles(), (file) => fs.readFileSync(path.join(REPO_ROOT, file), "utf8"));
  requireMatches("entries of the libraries", entries);
  failOnFindings("An entry of a library reaches a testing module", findings);
}

export const TESTING_ENTRY_VERIFIERS = [
  {
    id: "no-testing-module-in-entry",
    scope: "libs",
    module: "testing-entries",
    name: "No Testing Module In An Entry (no module in a testing folder is reachable by relative imports from an entry of a package under libs/ that is not itself in a testing folder)",
    run: verifyNoTestingInEntries,
  },
];
