import { spawn } from "node:child_process";
import { listGoModules } from "./modules.mjs";

const MAX_PARALLEL_MODULES = 4;

/**
 * Runs golangci-lint with the configuration of the module itself.
 *
 * @param {{ name: string, dir: string }} module
 * @returns {Promise<string[]>}
 */
function lintModule({ name, dir }) {
  return new Promise((resolve) => {
    const child = spawn("golangci-lint", ["run", "--allow-parallel-runners", "./..."], {
      cwd: dir,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let output = "";
    child.stdout.on("data", (d) => {
      output += d.toString();
    });
    child.stderr.on("data", (d) => {
      output += d.toString();
    });
    child.on("error", (error) => resolve([`[${name}] cannot run golangci-lint: ${error.message}`]));
    child.on("close", () => {
      const issues = output
        .split("\n")
        .map((l) => l.trim())
        .filter((l) => /^\S+\.go:\d+:\d+: /.test(l))
        // a package that cannot be loaded here (a cgo library the machine lacks) is not a finding
        .filter((l) => !l.includes("could not import"));
      resolve(issues.map((l) => `[${name}] ${l}`));
    });
  });
}

/**
 * Runs the golangci-lint configuration of each Go module, which for a plugin is its own
 * .golangci.yml, and reports every finding. Only the modules holding a changed directory are
 * checked when `targetDirs` is given.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean, targetDirs?: string[] | null }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runGolintCheck({ repoRoot, warnOnly = false, targetDirs = null }) {
  const start = Date.now();
  const modules = listGoModules(repoRoot).filter(
    ({ name }) =>
      !targetDirs || targetDirs.some((dir) => dir === name || dir.startsWith(`${name}/`)),
  );

  const issues = [];
  for (let i = 0; i < modules.length; i += MAX_PARALLEL_MODULES) {
    const batch = modules.slice(i, i + MAX_PARALLEL_MODULES);
    issues.push(...(await Promise.all(batch.map(lintModule))).flat());
  }

  const durationMs = Date.now() - start;
  return {
    id: "golint",
    name: "Go Lint (golangci-lint, each module's own configuration)",
    pass: issues.length === 0 || warnOnly,
    summary:
      issues.length === 0
        ? `Zero findings across ${modules.length} Go module(s)`
        : `${issues.length} finding(s) across ${modules.length} Go module(s)`,
    details: issues.length > 0 ? issues.join("\n") : undefined,
    durationMs,
  };
}
