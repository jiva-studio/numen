import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * Runs copy-paste detection across the monorepo using jscpd asynchronously.
 *
 * @param {{ repoRoot: string, threshold?: number }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runDuplicationCheck({ repoRoot, threshold = 3.5 }) {
  const start = Date.now();
  const binary = path.join(repoRoot, "node_modules/.bin/jscpd");
  const hasLocal = fs.existsSync(binary);
  const cmd = hasLocal ? binary : "npx";
  const args = hasLocal ? [] : ["--yes", "jscpd"];

  const outDir = path.join(repoRoot, "tools/lint-quality/.jscpd-report");

  args.push(
    "libs",
    "apps",
    "plugins",
    "--ignore",
    "**/*.test.*,**/*_test.go,**/*.spec.*,**/*.stories.*,**/fixtures/**,**/testdata/**,**/node_modules/**,**/dist/**,**/.claude/**,**/gen/**,**/*.yml,**/*.yaml,**/Makefile,**/go.mod,**/go.sum",
    "--reporters",
    "json",
    "--output",
    outDir
  );

  return new Promise((resolve) => {
    const child = spawn(cmd, args, {
      cwd: repoRoot,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (d) => {
      stdout += d.toString();
    });
    child.stderr.on("data", (d) => {
      stderr += d.toString();
    });

    child.on("close", (code) => {
      const durationMs = Date.now() - start;
      const reportPath = path.join(outDir, "jscpd-report.json");

      if (fs.existsSync(reportPath)) {
        try {
          const data = JSON.parse(fs.readFileSync(reportPath, "utf8"));
          fs.rmSync(outDir, { recursive: true, force: true });

          const total = data.statistics?.total || {};
          const percentNum = Number(total.percentage) || 0;
          const clones = total.clones || 0;
          const duplicatedLines = total.duplicatedLines || 0;
          const pass = percentNum <= threshold;

          resolve({
            id: "duplication",
            name: "Code Duplication (jscpd)",
            pass,
            summary: `${percentNum.toFixed(2)}% duplicated lines (${clones} clones, ${duplicatedLines} lines, max allowed: ${threshold}%)`,
            details: pass
              ? undefined
              : `Duplication rate ${percentNum.toFixed(2)}% exceeds threshold of ${threshold}%.`,
            durationMs,
          });
          return;
        } catch {
          // Fallback to exit code
        }
      }

      const pass = code === 0;
      resolve({
        id: "duplication",
        name: "Code Duplication (jscpd)",
        pass,
        summary: pass ? "Duplication within acceptable limits" : "Duplication check failed",
        details: stderr || stdout || undefined,
        durationMs,
      });
    });
  });
}
