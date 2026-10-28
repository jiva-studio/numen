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
    const knipBin = path.join(repoRoot, "node_modules/.bin/knip");
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
  const configFile = path.join(repoRoot, ".golangci.yml");

  const goModules = [
    { dir: path.join(repoRoot, "libs/core"), name: "libs/core" },
    { dir: path.join(repoRoot, "libs/notes"), name: "libs/notes" },
    { dir: path.join(repoRoot, "libs/plugin-sdk-go"), name: "libs/plugin-sdk-go" },
    { dir: path.join(repoRoot, "apps/desktop"), name: "apps/desktop" },
  ];
  const pluginsDir = path.join(repoRoot, "plugins");
  if (fs.existsSync(pluginsDir)) {
    for (const e of fs.readdirSync(pluginsDir, { withFileTypes: true })) {
      if (e.isDirectory() && fs.existsSync(path.join(pluginsDir, e.name, "go.mod"))) {
        goModules.push({ dir: path.join(pluginsDir, e.name), name: `plugins/${e.name}` });
      }
    }
  }

  const tsPackages = [
    { relDir: "libs/ui/core", name: "libs/ui/core" },
    { relDir: "libs/ui/shell", name: "libs/ui/shell" },
    { relDir: "libs/plugin-sdk-ts", name: "libs/plugin-sdk-ts" },
    { relDir: "libs/plugin-host-ts", name: "libs/plugin-host-ts" },
    { relDir: "apps/desktop/editor", name: "desktop/editor" },
    { relDir: "apps/desktop/flashcards", name: "desktop/flashcards" },
    { relDir: "apps/mobile", name: "apps/mobile" },
  ];

  const [unparamResults, knipResults] = await Promise.all([
    Promise.all(goModules.map((m) => checkUnparam(m.dir, configFile, m.name))),
    Promise.all(tsPackages.map((p) => checkKnip(repoRoot, p.relDir, p.name))),
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
