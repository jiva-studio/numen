import { spawn } from "node:child_process";
import { findGoLintConfig, listGoModules } from "./modules.mjs";

/**
 * Runs golangci-lint on a single directory.
 *
 * @param {string} dir
 * @param {string} configFile
 * @param {string} pluginName
 * @returns {Promise<string[]>}
 */
function lintPlugin(dir, configFile, pluginName) {
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
        "cyclop,gocognit,nestif,funlen",
        "./...",
      ],
      {
        cwd: dir,
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
        .filter((l) => l.length > 0 && !l.startsWith("level=") && !l.includes("issues:"));

      const issues = [];
      for (const line of lines) {
        if (line.includes("(") && line.includes(")")) {
          issues.push(`[${pluginName}] ${line}`);
        }
      }
      resolve(issues);
    });
  });
}

/**
 * Checks cyclomatic and cognitive complexity in Go code asynchronously across all modules.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runComplexityCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const targets = listGoModules(repoRoot);

  const results = await Promise.all(
    targets.map(({ dir, name }) => lintPlugin(dir, findGoLintConfig(repoRoot, dir, ".golangci-structure.yml"), name))
  );

  const allIssues = results.flat();
  const durationMs = Date.now() - start;
  const pass = allIssues.length === 0 || warnOnly;
  const summary =
    allIssues.length === 0
      ? `Zero complexity violations across ${targets.length} Go modules`
      : `${allIssues.length} complexity violations across Go modules`;

  return {
    id: "complexity",
    name: "Cyclomatic & Cognitive Complexity (Go)",
    pass,
    summary,
    details: allIssues.length > 0 ? allIssues.join("\n") : undefined,
    durationMs,
  };
}
