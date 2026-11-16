import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * Runs knip with the root knip.json in repoRoot and resolves to its findings, one line each, or to none
 * when it exits with 0.
 *
 * @param {string} repoRoot
 * @param {string[]} args
 * @returns {Promise<string[]>}
 */
export function runKnip(repoRoot, args) {
  const knipBin = path.join(repoRoot, "node_modules/.bin/knip");
  if (!fs.existsSync(knipBin)) {
    return Promise.resolve(["[TS knip] node_modules/.bin/knip is missing; run npm ci"]);
  }
  if (!fs.existsSync(path.join(repoRoot, "knip.json"))) {
    return Promise.resolve(["[TS knip] knip.json is missing at the repository root"]);
  }
  return new Promise((resolve) => {
    const child = spawn(knipBin, ["--no-progress", "--reporter", "compact", ...args], {
      cwd: repoRoot,
      stdio: ["ignore", "pipe", "pipe"],
    });
    let output = "";
    child.stdout.on("data", (d) => (output += d.toString()));
    child.stderr.on("data", (d) => (output += d.toString()));
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
 * Fails on every export and type that no shipped file uses. knip runs in production mode, which reads
 * only the entries and project files marked with `!` in knip.json, so a test, a story or a test helper
 * is no user of an export.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runTestOnlyExportsCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const issues = await runKnip(repoRoot, ["--production", "--include", "exports,types"]);
  return {
    id: "test-only-exports",
    name: "Exports Used Only by Tests (knip --production)",
    pass: issues.length === 0 || warnOnly,
    summary: issues.length === 0 ? "Every export has a shipped user" : `${issues.length} knip finding(s)`,
    details: issues.length > 0 ? issues.join("\n") : undefined,
    durationMs: Date.now() - start,
  };
}
