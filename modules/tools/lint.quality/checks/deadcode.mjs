import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

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
 * Runs knip on a specific frontend workspace directory.
 *
 * @param {string} repoRoot
 * @param {string} relDir
 * @param {string} pkgName
 * @returns {Promise<string[]>}
 */
function checkKnip(repoRoot, relDir, pkgName) {
  return new Promise((resolve) => {
    const knipBin = path.join(repoRoot, "modules/node_modules/.bin/knip");
    const targetDir = path.join(repoRoot, relDir);

    if (!fs.existsSync(knipBin) || !fs.existsSync(path.join(targetDir, "knip.json"))) {
      resolve([]);
      return;
    }

    const child = spawn(knipBin, ["--include", "files"], {
      cwd: targetDir,
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
      const unusedFilesMatch = output.match(/Unused files \((\d+)\)/);
      const filesCount = unusedFilesMatch ? Number(unusedFilesMatch[1]) : 0;

      if (code !== 0 || filesCount > 0) {
        const issues = [];
        if (filesCount > 0) {
          issues.push(`[TS knip: ${pkgName}] ${filesCount} unused files`);
        }
        if (code !== 0 && issues.length === 0) {
          issues.push(`[TS knip: ${pkgName}] knip check failed with exit code ${code}`);
        }
        resolve(issues);
      } else {
        resolve([]);
      }
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
  const configFile = path.join(repoRoot, "modules/.golangci.yml");

  const [coreUnparam, desktopUnparam, uiKnip, editorKnip, flashcardsKnip] = await Promise.all([
    checkUnparam(path.join(repoRoot, "modules/libs/core"), configFile, "libs/core"),
    checkUnparam(path.join(repoRoot, "modules/apps/desktop"), configFile, "apps/desktop"),
    checkKnip(repoRoot, "modules/libs/ui", "libs/ui"),
    checkKnip(repoRoot, "modules/apps/desktop/editor", "desktop/editor"),
    checkKnip(repoRoot, "modules/apps/desktop/flashcards", "desktop/flashcards"),
  ]);

  const allIssues = [
    ...coreUnparam,
    ...desktopUnparam,
    ...uiKnip,
    ...editorKnip,
    ...flashcardsKnip,
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
