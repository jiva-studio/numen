import fs from "node:fs";
import path from "node:path";

import { runKnip } from "../../checks/structure/ts/test-only-exports/check.mjs";
import { listFrontendPackages } from "./modules.mjs";

/**
 * Lists the frontend packages that no workspace of the root knip.json names.
 * A key is a directory or a directory followed by `/*`.
 *
 * @param {string[]} workspaceKeys
 * @param {string[]} packageDirs
 * @returns {string[]}
 */
export function findPackagesWithoutKnip(workspaceKeys, packageDirs) {
  const isNamed = (dir) =>
    workspaceKeys.some((key) =>
      key.endsWith("/*") ? path.posix.dirname(dir) === key.slice(0, -2) : key === dir,
    );
  return packageDirs.filter((dir) => !isNamed(dir));
}

/**
 * Runs knip once over every workspace of the root knip.json, tests included, for unused files and
 * dependencies. Findings at the level `error` in its rules fail the check; a missing binary, a missing
 * config or a frontend package the config does not name fail it as well. Unused exports are the check
 * `ts/test-only-exports` of tools/checks.
 *
 * @param {string} repoRoot
 * @param {string[]} packageDirs
 * @returns {Promise<string[]>}
 */
async function checkKnip(repoRoot, packageDirs) {
  const configFile = path.join(repoRoot, "knip.json");
  if (!fs.existsSync(configFile)) {
    return ["[TS knip] knip.json is missing at the repository root"];
  }
  const workspaceKeys = Object.keys(JSON.parse(fs.readFileSync(configFile, "utf8")).workspaces ?? {});
  const uncovered = findPackagesWithoutKnip(workspaceKeys, packageDirs).map(
    (dir) => `[TS knip] ${dir} is not a workspace of knip.json`,
  );
  const found = await runKnip(repoRoot, ["--include", "files,dependencies,unlisted,binaries,unresolved"]);
  return [...uncovered, ...found];
}

/**
 * Checks the TypeScript and Vue packages for unused files and dependencies with knip.
 * Unused Go parameters and results are the `unparam` linter of .golangci-structure.yml.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runKnipCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const packages = listFrontendPackages(repoRoot);
  const issues = await checkKnip(repoRoot, packages.map((p) => p.relDir));

  return {
    id: "knip",
    name: "Unused Files and Dependencies (knip)",
    pass: issues.length === 0 || warnOnly,
    summary: issues.length === 0 ? "Zero knip findings" : `${issues.length} knip finding(s)`,
    details: issues.length > 0 ? issues.join("\n") : undefined,
    durationMs: Date.now() - start,
  };
}
