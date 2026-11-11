import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { runTool } from "../../shared/tools.mjs";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

/** The smallest number of TypeScript configurations the repository has; fewer means discovery broke. */

/**
 * The `exclude` entries of a TypeScript configuration, each with its line, read from its JSON syntax tree.
 *
 * @param {string} file The configuration's name, for the syntax tree
 * @param {string} text The configuration's text
 * @returns {Array<{ line: number, value: string }>}
 */
export function readExcludeEntries(file, text) {
  const source = ts.parseJsonText(file, text);
  const root = source.statements[0]?.expression;
  if (!root || !ts.isObjectLiteralExpression(root)) return [];
  const entries = [];
  for (const property of root.properties) {
    if (!ts.isPropertyAssignment(property) || !ts.isStringLiteral(property.name) || property.name.text !== "exclude") continue;
    if (!ts.isArrayLiteralExpression(property.initializer)) continue;
    for (const element of property.initializer.elements) {
      if (!ts.isStringLiteral(element)) continue;
      const line = source.getLineAndCharacterOfPosition(element.getStart(source)).line + 1;
      entries.push({ line, value: element.text });
    }
  }
  return entries;
}

/** Whether a repository-relative pattern names a file: the file itself, a folder that holds it, or a glob over it. */
function isNamedBy(pattern, file) {
  if (file === pattern || file.startsWith(`${pattern}/`)) return true;
  if (typeof path.posix.matchesGlob !== "function") return false;
  return path.posix.matchesGlob(file, pattern) || path.posix.matchesGlob(file, `${pattern}/**`);
}

/**
 * The `exclude` entries that name a tracked file. What a build writes (`dist`, `node_modules`) is not tracked, so an
 * entry that names a tracked file leaves a source out of the typecheck.
 *
 * @param {Array<{ file: string, entries: Array<{ line: number, value: string }> }>} configs Repository-relative
 *   configurations and their entries
 * @param {string[]} trackedFiles Repository-relative paths of the tracked files
 * @returns {Array<{ file: string, line: number, symbol: string }>}
 */
export function findExcludedSources(configs, trackedFiles) {
  const findings = [];
  for (const { file, entries } of configs) {
    const dir = path.posix.dirname(file);
    for (const { line, value } of entries) {
      let pattern = path.posix.join(dir, value);
      if (pattern.endsWith("/")) pattern = pattern.slice(0, -1);
      const named = trackedFiles.find((one) => isNamedBy(pattern, one));
      if (named) findings.push({ file, line, symbol: `exclude "${value}" leaves out ${named}` });
    }
  }
  return findings;
}

/** Whether a repository-relative file is a TypeScript configuration (`tsconfig.json`, `tsconfig.app.json`, ...). */
export function isTsConfig(file) {
  const name = path.posix.basename(file);
  return name.startsWith("tsconfig") && name.endsWith(".json");
}

function readTrackedFiles() {
  const listed = runTool("git", ["ls-files"], { cwd: REPO_ROOT, encoding: "utf8", maxBuffer: 64 * 1024 * 1024 });
  if (listed.status !== 0) throw new Error(`git ls-files failed: ${listed.stderr}`);
  return listed.stdout.split("\n").filter((file) => file && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** No TypeScript configuration excludes a tracked file. */
export function verifyTypecheckCoversSources() {
  const tracked = readTrackedFiles();
  const configs = tracked
    .filter(isTsConfig)
    .map((file) => ({ file, entries: readExcludeEntries(file, fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) }));
  requireMatches("TypeScript configurations", configs.length);
  failOnFindings("A TypeScript configuration excludes a source file", findExcludedSources(configs, tracked));
}

export const TYPECHECK_SCOPE_VERIFIERS = [
  {
    id: "typecheck-covers-sources",
    scope: "libs",
    module: "lint",
    name: "No TypeScript Configuration Excludes a Tracked File (only build output is excluded)",
    run: verifyTypecheckCoversSources,
  },
];
