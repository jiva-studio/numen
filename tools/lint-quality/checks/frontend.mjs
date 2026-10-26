import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * Runs ESLint on a single frontend target directory using JSON output.
 *
 * @param {string} repoRoot
 * @param {string} relDir
 * @param {string} pkgName
 * @returns {Promise<{ pkgName: string, errorCount: number, issues: string[], ruleStats: Record<string, number> }>}
 */
function lintPackage(repoRoot, relDir, pkgName) {
  return new Promise((resolve) => {
    const eslintBin = path.join(repoRoot, "node_modules/.bin/eslint");
    const targetDir = path.join(repoRoot, relDir);

    if (!fs.existsSync(eslintBin) || !fs.existsSync(targetDir)) {
      resolve({ pkgName, errorCount: 0, issues: [], ruleStats: {} });
      return;
    }

    const child = spawn(eslintBin, ["-f", "json", "."], {
      cwd: targetDir,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let stdout = "";
    child.stdout.on("data", (d) => {
      stdout += d.toString();
    });

    child.on("close", () => {
      try {
        const results = JSON.parse(stdout);
        const issues = [];
        const ruleStats = {};
        let errorCount = 0;

        for (const file of results) {
          const relFile = path.relative(repoRoot, file.filePath);
          for (const msg of file.messages) {
            if (msg.severity === 2) {
              errorCount++;
              const ruleId = msg.ruleId || "fatal-error";
              ruleStats[ruleId] = (ruleStats[ruleId] || 0) + 1;
              issues.push(
                `[${pkgName}] ${relFile}:${msg.line}:${msg.column} - ${msg.message} (${ruleId})`
              );
            }
          }
        }

        resolve({ pkgName, errorCount, issues, ruleStats });
      } catch {
        resolve({ pkgName, errorCount: 0, issues: [], ruleStats: {} });
      }
    });
  });
}

/**
 * Checks frontend code quality and conventions across Vue and TypeScript packages.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runFrontendCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();

  const targets = [
    { relDir: "libs/ui/core", name: "libs/ui/core" },
    { relDir: "apps/desktop/editor", name: "desktop/editor" },
    { relDir: "apps/desktop/flashcards", name: "desktop/flashcards" },
    { relDir: "apps/mobile", name: "mobile" },
    { relDir: "plugins", name: "plugins" },
  ];

  const results = await Promise.all(
    targets.map((t) => lintPackage(repoRoot, t.relDir, t.name))
  );

  const durationMs = Date.now() - start;
  const totalErrors = results.reduce((acc, r) => acc + r.errorCount, 0);

  const allRuleStats = {};
  for (const r of results) {
    for (const [rule, count] of Object.entries(r.ruleStats)) {
      allRuleStats[rule] = (allRuleStats[rule] || 0) + count;
    }
  }

  const topRules = Object.entries(allRuleStats)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 5)
    .map(([rule, count]) => `${rule} (${count})`)
    .join(", ");

  const perPackageSummary = results
    .map((r) => `[${r.pkgName}] ${r.errorCount} error(s)`)
    .join(", ");

  const summary =
    totalErrors === 0
      ? "Zero frontend quality violations across all TS/Vue packages"
      : `${totalErrors} frontend violations across packages: ${perPackageSummary}${topRules ? ` | Top: ${topRules}` : ""}`;

  const allIssues = results.flatMap((r) => r.issues);
  const pass = totalErrors === 0 || warnOnly;

  return {
    id: "frontend",
    name: "Frontend Conventions & Quality (ESLint)",
    pass,
    summary,
    details: allIssues.length > 0 ? allIssues.join("\n") : undefined,
    durationMs,
  };
}
