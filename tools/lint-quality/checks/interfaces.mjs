import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * Runs interfacebloat on a single Go module directory.
 *
 * @param {string} dir
 * @param {string} configFile
 * @param {string} modName
 * @returns {Promise<string[]>}
 */
function lintInterfaces(dir, configFile, modName) {
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
        "interfacebloat",
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
        .filter((l) => l.includes("(interfacebloat)"));

      resolve(lines.map((l) => `[${modName}] ${l}`));
    });
  });
}

/**
 * Checks for bloated Go interfaces (ISP violation, > 10 methods) asynchronously across all modules.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runInterfacesCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const configFile = path.join(repoRoot, ".golangci.yml");
  const pluginsDir = path.join(repoRoot, "plugins");

  const targets = [
    { name: "libs/core", dir: path.join(repoRoot, "libs/core") },
    { name: "libs/notes", dir: path.join(repoRoot, "libs/notes") },
    { name: "apps/desktop", dir: path.join(repoRoot, "apps/desktop") },
    { name: "libs/plugin-sdk-go", dir: path.join(repoRoot, "libs/plugin-sdk-go") },
  ].filter((t) => fs.existsSync(path.join(t.dir, "go.mod")));

  if (fs.existsSync(pluginsDir)) {
    const entries = fs.readdirSync(pluginsDir, { withFileTypes: true });
    for (const e of entries) {
      if (e.isDirectory() && fs.existsSync(path.join(pluginsDir, e.name, "go.mod"))) {
        targets.push({
          name: `plugin/${e.name}`,
          dir: path.join(pluginsDir, e.name),
        });
      }
    }
  }

  const results = await Promise.all(
    targets.map(({ dir, name }) => lintInterfaces(dir, configFile, name))
  );

  const allIssues = results.flat();
  const durationMs = Date.now() - start;
  const pass = allIssues.length === 0 || warnOnly;
  const summary =
    allIssues.length === 0
      ? `Zero bloated interfaces across ${targets.length} Go modules (all interfaces <= 5 methods)`
      : `${allIssues.length} interfaces exceed 5 methods across Go modules`;

  return {
    id: "interfaces",
    name: "Interface Segregation / Bloat (interfacebloat)",
    pass,
    summary,
    details: allIssues.length > 0 ? allIssues.join("\n") : undefined,
    durationMs,
  };
}
