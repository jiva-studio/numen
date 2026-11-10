import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

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
 * Runs knip once over every workspace of the root knip.json. Findings at the
 * level `error` in its rules fail the check; a missing binary, a missing
 * config or a frontend package the config does not name fail it as well.
 *
 * @param {string} repoRoot
 * @param {string[]} packageDirs
 * @returns {Promise<string[]>}
 */
function checkKnip(repoRoot, packageDirs) {
  const knipBin = path.join(repoRoot, "node_modules/.bin/knip");
  const configFile = path.join(repoRoot, "knip.json");
  if (!fs.existsSync(knipBin)) {
    return Promise.resolve(["[TS knip] node_modules/.bin/knip is missing; run npm ci"]);
  }
  if (!fs.existsSync(configFile)) {
    return Promise.resolve(["[TS knip] knip.json is missing at the repository root"]);
  }
  const workspaceKeys = Object.keys(JSON.parse(fs.readFileSync(configFile, "utf8")).workspaces ?? {});
  const uncovered = findPackagesWithoutKnip(workspaceKeys, packageDirs).map(
    (dir) => `[TS knip] ${dir} is not a workspace of knip.json`,
  );

  return Promise.all(KNIP_RUNS.map((run) => runKnip(knipBin, repoRoot, run))).then((lists) => [...uncovered, ...lists.flat()]);
}

/**
 * The two runs of knip. Dead code is judged on what ships: the production run reads no test, story or
 * test helper as a user of a file or an export. Dependencies are judged on everything, tests included.
 */
const KNIP_RUNS = [
  ["--production", "--include", "exports,types"],
  ["--include", "files,dependencies,unlisted,binaries,unresolved"],
];

function runKnip(knipBin, repoRoot, run) {
  return new Promise((resolve) => {
    const child = spawn(knipBin, ["--no-progress", "--reporter", "compact", ...run], {
      cwd: repoRoot,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let output = "";
    child.stdout.on("data", (d) => {
      output += d.toString();
    });
    child.stderr.on("data", (d) => {
      output += d.toString();
    });

    child.on("close", (code) => {
      if (code === 0) {
        resolve([]);
        return;
      }
      const lines = output
        .split("\n")
        .map((l) => l.trim())
        .filter(Boolean)
        .map((l) => `[TS knip] ${l}`);
      resolve(lines.length > 0 ? lines : [`[TS knip] knip exited with code ${code}`]);
    });
  });
}

/**
 * Checks the TypeScript and Vue packages for unused files, dependencies and exports with knip.
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
    name: "Unused Files, Dependencies and Exports (knip)",
    pass: issues.length === 0 || warnOnly,
    summary: issues.length === 0 ? "Zero knip findings" : `${issues.length} knip finding(s)`,
    details: issues.length > 0 ? issues.join("\n") : undefined,
    durationMs: Date.now() - start,
  };
}
