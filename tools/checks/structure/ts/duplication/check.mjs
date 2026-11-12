import { spawn } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const IGNORE =
  "**/*.test.*,**/*_test.go,**/*.spec.*,**/*.stories.*,**/fixtures/**,**/testdata/**,**/node_modules/**,**/dist/**,**/.claude/**,**/gen/**,**/*.yml,**/*.yaml,**/Makefile,**/go.mod,**/go.sum";

const CONCURRENCY = 4;

/**
 * Lists the duplication scopes. The host code (libs, apps) is one scope;
 * each plugin directory is its own scope, because plugins are separate
 * repositories that depend only on the SDK and never share files.
 */
function listScopes(repoRoot) {
  const scopes = [{ name: "host", paths: ["libs", "apps"] }];
  const pluginsDir = path.join(repoRoot, "plugins");
  for (const entry of fs.readdirSync(pluginsDir, { withFileTypes: true })) {
    if (!entry.isDirectory() || entry.name.startsWith(".")) continue;
    scopes.push({ name: `plugins/${entry.name}`, paths: [`plugins/${entry.name}`] });
  }
  return scopes;
}

function runJscpd({ repoRoot, scope, outDir }) {
  const binary = path.join(repoRoot, "node_modules/.bin/jscpd");
  if (!fs.existsSync(binary)) {
    return Promise.resolve({ scope: scope.name, error: "node_modules/.bin/jscpd is missing; run npm ci" });
  }
  const args = [...scope.paths, "--ignore", IGNORE, "--reporters", "json", "--output", outDir];

  return new Promise((resolve) => {
    const child = spawn(binary, args, { cwd: repoRoot, stdio: ["ignore", "pipe", "pipe"] });
    let output = "";
    child.stdout.on("data", (d) => (output += d.toString()));
    child.stderr.on("data", (d) => (output += d.toString()));
    child.on("close", (code) => {
      const reportPath = path.join(outDir, "jscpd-report.json");
      if (!fs.existsSync(reportPath)) {
        resolve({ scope: scope.name, error: `jscpd exited with code ${code}: ${output}` });
        return;
      }
      try {
        const data = JSON.parse(fs.readFileSync(reportPath, "utf8"));
        const total = data.statistics?.total || {};
        resolve({
          scope: scope.name,
          percent: Number(total.percentage) || 0,
          clones: total.clones || 0,
          lines: total.duplicatedLines || 0,
        });
      } catch (err) {
        resolve({ scope: scope.name, error: `unreadable jscpd report: ${err.message}` });
      }
    });
  });
}

/**
 * Runs copy-paste detection with jscpd, once for the host code and once per
 * plugin. Each scope must stay within the threshold.
 *
 * @param {{ repoRoot: string, threshold?: number }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runDuplicationCheck({ repoRoot, threshold = 3.5 }) {
  const start = Date.now();
  const scopes = listScopes(repoRoot);
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), "jscpd-"));
  const results = [];
  let next = 0;

  async function worker(id) {
    while (next < scopes.length) {
      const scope = scopes[next++];
      const outDir = path.join(tmpRoot, `${id}-${next}`);
      results.push(await runJscpd({ repoRoot, scope, outDir }));
    }
  }

  await Promise.all(Array.from({ length: CONCURRENCY }, (_, id) => worker(id)));
  fs.rmSync(tmpRoot, { recursive: true, force: true });

  const failures = results.filter((r) => r.error || r.percent > threshold);
  const pass = failures.length === 0;
  const worst = results.reduce((a, b) => ((b.percent || 0) > (a.percent || 0) ? b : a), { scope: "none", percent: 0 });
  const lines = failures.map((r) =>
    r.error
      ? `${r.scope}: ${r.error}`
      : `${r.scope}: ${r.percent.toFixed(2)}% exceeds threshold of ${threshold}% (${r.clones} clones, ${r.lines} lines)`,
  );

  return {
    id: "duplication",
    name: "Code Duplication (jscpd)",
    pass,
    summary: `${scopes.length} scopes (host and each plugin), worst ${worst.scope} at ${(worst.percent || 0).toFixed(2)}% (max allowed: ${threshold}%)`,
    details: pass ? undefined : lines.join("\n"),
    durationMs: Date.now() - start,
  };
}
