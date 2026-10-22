import { spawn } from "node:child_process";
import path from "node:path";

/**
 * Checks for bloated Go interfaces (ISP violation, > 10 methods) asynchronously.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runInterfacesCheck({ repoRoot, warnOnly = false }) {
  const start = Date.now();
  const configFile = path.join(repoRoot, "modules/.golangci.yml");
  const mod = "modules/libs/core";
  const cwd = path.join(repoRoot, mod);

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
        .filter((l) => l.includes("(interfacebloat)"));

      const durationMs = Date.now() - start;
      const pass = lines.length === 0 || warnOnly;
      const summary =
        lines.length === 0
          ? "Zero bloated interfaces (all interfaces <= 10 methods)"
          : `${lines.length} interfaces exceed 10 methods`;

      resolve({
        id: "interfaces",
        name: "Interface Segregation / Bloat (interfacebloat)",
        pass,
        summary,
        details: lines.length > 0 ? lines.join("\n") : undefined,
        durationMs,
      });
    });
  });
}
