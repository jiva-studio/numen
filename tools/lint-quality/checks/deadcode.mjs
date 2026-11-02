import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

import { listFrontendPackages, listGoModules } from "./modules.mjs";

/**
 * Runs unparam on a single Go module.
 *
 * @param {string} cwd
 * @param {string} configFile
 * @param {string} modName
 * @returns {Promise<string[]>}
 */
function checkUnparam(cwd, configFile, modName) {
  return new Promise((resolve) => {
    const child = spawn(
      "golangci-lint",
      [
        "run",
        "--config",
        configFile,
        "--tests=false",
        "--allow-parallel-runners",
        "--enable-only",
        "unparam",
        "./...",
      ],
      {
        cwd,
        stdio: ["ignore", "pipe", "pipe"],
      }
    );

    let output = "";
    child.stdout.on("data", (d) => {
      output += d.toString();
    });
    child.stderr.on("data", (d) => {
      output += d.toString();
    });

    child.on("close", () => {
      const lines = output
        .split("\n")
        .map((l) => l.trim())
        .filter((l) => l.includes("(unparam)"));
      resolve(lines.map((l) => `[Go unparam: ${modName}] ${l}`));
    });
  });
}

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

  return new Promise((resolve) => {
    const child = spawn(knipBin, ["--no-progress", "--reporter", "compact"], {
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
        resolve(uncovered);
        return;
      }
      const lines = output
        .split("\n")
        .map((l) => l.trim())
        .filter(Boolean)
        .map((l) => `[TS knip] ${l}`);
      resolve([...uncovered, ...(lines.length > 0 ? lines : [`[TS knip] knip exited with code ${code}`])]);
    });
  });
}

/**
 * Checks for dead code, unused exports, and unread parameters asynchronously.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runDeadcodeCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const configFile = path.join(repoRoot, ".golangci.yml");

  const goModules = listGoModules(repoRoot);
  const tsPackages = listFrontendPackages(repoRoot);

  const [unparamResults, knipResults] = await Promise.all([
    Promise.all(goModules.map((m) => checkUnparam(m.dir, configFile, m.name))),
    checkKnip(repoRoot, tsPackages.map((p) => p.relDir)),
  ]);

  const allIssues = [
    ...unparamResults.flat(),
    ...knipResults.flat(),
  ];
  const durationMs = Date.now() - start;
  const pass = allIssues.length === 0 || warnOnly;
  const summary =
    allIssues.length === 0
      ? "Zero unused parameters in Go and zero dead exports in UI"
      : `${allIssues.length} dead code / unused symbol findings`;

  return {
    id: "deadcode",
    name: "Dead Code & Unused Symbols (knip & unparam)",
    pass,
    summary,
    details: allIssues.length > 0 ? allIssues.join("\n") : undefined,
    durationMs,
  };
}
