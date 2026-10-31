#!/usr/bin/env node
/**
 * Numen Code Quality & Metrics Validator (@numen/lint-quality)
 *
 * Validates duplication, cognitive & cyclomatic complexity, dead code,
 * and interface sizing across the monorepo.
 *
 * Usage:
 *   node check.mjs [options]
 *   node check.mjs --check=duplication
 *   node check.mjs --changed-only
 *   node check.mjs --help
 */
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import process from "node:process";

import { runDuplicationCheck } from "./checks/duplication.mjs";
import { runComplexityCheck } from "./checks/complexity.mjs";
import { runDeadcodeCheck } from "./checks/deadcode.mjs";
import { runInterfacesCheck } from "./checks/interfaces.mjs";
import { runGolintCheck } from "./checks/golint.mjs";
import { runFrontendCheck } from "./checks/frontend.mjs";
import { runStructureCheck } from "./checks/structure.mjs";

const SCRIPT_DIR = path.dirname(new URL(import.meta.url).pathname);
const REPO_ROOT = path.resolve(SCRIPT_DIR, "../..");

const ALL_CHECKS = [
  {
    id: "duplication",
    name: "Code Duplication (jscpd)",
    run: (opts) => runDuplicationCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "complexity",
    name: "Cyclomatic & Cognitive Complexity (Go)",
    run: (opts) => runComplexityCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "deadcode",
    name: "Dead Code & Unused Symbols (knip & unparam)",
    run: (opts) => runDeadcodeCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "interfaces",
    name: "Interface Segregation / Bloat (interfacebloat)",
    run: (opts) => runInterfacesCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "golint",
    name: "Go Lint (golangci-lint, each module's own configuration)",
    run: (opts) => runGolintCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "frontend",
    name: "Frontend Conventions & Quality (ESLint)",
    run: (opts) => runFrontendCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
  {
    id: "structure",
    name: "Directory Structure & Size (max 20 source files)",
    run: (opts) => runStructureCheck({ repoRoot: REPO_ROOT, ...opts }),
  },
];

function printHelp() {
  console.log(`
Usage: node check.mjs [options]

Options:
  --check=<name>       Run specific check(s) by ID (comma-separated: duplication,complexity,deadcode,interfaces,golint,frontend,structure)
  --changed-only       Run only checks relevant to modified files against git base
  --warn               Non-blocking mode: report findings with exit code 0
  --strict             Fail on any detected finding (default)
  --json               Output machine-readable JSON results
  --github-summary     Write markdown summary to $GITHUB_STEP_SUMMARY
  --list               List available checks
  --help, -h           Show this help message
`);
}

function getChangedFiles() {
  const currentBranch = spawnSync("git", ["rev-parse", "--abbrev-ref", "HEAD"], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  }).stdout?.trim();

  const uncommitted =
    spawnSync("git", ["diff", "--name-only", "HEAD"], {
      cwd: REPO_ROOT,
      encoding: "utf8",
    }).stdout?.split("\n").map((f) => f.trim()).filter(Boolean) || [];

  let base = process.env.BASE_REF;
  if (!base) {
    base = currentBranch === "main" ? "HEAD~1" : "origin/main";
  }

  let res = spawnSync("git", ["diff", "--name-only", `${base}...HEAD`], {
    cwd: REPO_ROOT,
    encoding: "utf8",
  });

  if (res.status !== 0) {
    res = spawnSync("git", ["diff", "--name-only", "HEAD~1...HEAD"], {
      cwd: REPO_ROOT,
      encoding: "utf8",
    });
  }

  const committed = (res.stdout || "").split("\n").map((f) => f.trim()).filter(Boolean);
  return Array.from(new Set([...uncommitted, ...committed]));
}

function resolveChecks(args, changedOnly) {
  let targetIds = ALL_CHECKS.map((c) => c.id);

  const checkArg = args.find((a) => a.startsWith("--check="));
  if (checkArg) {
    const raw = checkArg.split("=")[1];
    targetIds = raw.split(",").map((s) => s.trim().toLowerCase());
  }

  if (changedOnly) {
    const changed = getChangedFiles();
    if (changed.length === 0) {
      return [];
    }
    const hasGo = changed.some((f) => f.endsWith(".go"));
    const hasTsOrVue = changed.some((f) => f.endsWith(".ts") || f.endsWith(".vue") || f.endsWith(".astro"));
    const hasLintConfig = changed.some((f) => /(^|\/)(eslint\.config\.js|base-eslint\.mjs|tsconfig\.json)$/.test(f));
    const hasGoPlugins = changed.some((f) => f.startsWith("plugins/") && f.endsWith(".go"));
    const hasUi = changed.some((f) => f.startsWith("libs/ui/core/"));
    const hasGoCore = changed.some((f) => f.startsWith("libs/core/") && f.endsWith(".go"));

    targetIds = targetIds.filter((id) => {
      if (id === "duplication") return hasGo || hasTsOrVue;
      if (id === "complexity") return hasGo;
      if (id === "deadcode") return hasUi || hasGoCore;
      if (id === "interfaces") return hasGo;
      if (id === "golint") return hasGo;
      if (id === "frontend") return hasTsOrVue || hasLintConfig;
      if (id === "structure") return hasGo || hasTsOrVue;
      return true;
    });
  }

  return ALL_CHECKS.filter((c) => targetIds.includes(c.id));
}

function writeGithubSummary(results, totalMs) {
  const file = process.env.GITHUB_STEP_SUMMARY;
  if (!file) return;

  const rows = results
    .map(
      (r) =>
        `| ${r.pass ? "✅ Pass" : "❌ Fail"} | **${r.name}** | ${r.summary} | ${(r.durationMs / 1000).toFixed(1)}s |`
    )
    .join("\n");

  const md = `### 📊 Code Quality & Metrics Report

| Status | Check | Summary | Duration |
|---|---|---|---|
${rows}

*Total duration: ${(totalMs / 1000).toFixed(1)}s*
`;

  fs.appendFileSync(file, md, "utf8");
}

async function main() {
  const args = process.argv.slice(2);

  if (args.includes("--help") || args.includes("-h")) {
    printHelp();
    process.exit(0);
  }

  if (args.includes("--list")) {
    console.log("Available checks:");
    for (const c of ALL_CHECKS) {
      console.log(`  - ${c.id.padEnd(14)} : ${c.name}`);
    }
    process.exit(0);
  }

  const isWarn = args.includes("--warn");
  const isJson = args.includes("--json");
  const isChangedOnly = args.includes("--changed-only");
  const writeSummary = args.includes("--github-summary") || !!process.env.GITHUB_STEP_SUMMARY;

  const selected = resolveChecks(args, isChangedOnly);

  if (selected.length === 0) {
    if (isJson) {
      console.log(JSON.stringify({ status: "skipped", checks: [] }, null, 2));
    } else {
      console.log("No relevant checks to run for changed files.");
    }
    process.exit(0);
  }

  if (!isJson) {
    console.log(`🔍 Running ${selected.length} quality check(s) in parallel...\n`);
  }

  const startTime = Date.now();
  const targetDirs = isChangedOnly
    ? Array.from(new Set(getChangedFiles().map((f) => path.dirname(f))))
    : null;
  const results = await Promise.all(
    selected.map((c) => c.run({ warnOnly: isWarn, targetDirs }))
  );
  const totalMs = Date.now() - startTime;

  if (isJson) {
    console.log(
      JSON.stringify(
        {
          totalDurationMs: totalMs,
          allPassed: results.every((r) => r.pass),
          results,
        },
        null,
        2
      )
    );
    process.exit(results.every((r) => r.pass) ? 0 : 1);
  }

  let failedCount = 0;

  for (const r of results) {
    const icon = r.pass ? "✔" : "✖";
    const dur = `(${(r.durationMs / 1000).toFixed(1)}s)`;
    console.log(`${icon} [${r.id}] ${r.name} ${dur}`);
    console.log(`   ${r.summary}`);

    if (!r.pass && r.details) {
      failedCount++;
      const indented = r.details
        .split("\n")
        .slice(0, 15)
        .map((l) => `     ${l}`)
        .join("\n");
      console.log(`${indented}`);
      if (r.details.split("\n").length > 15) {
        console.log(`     ... and ${r.details.split("\n").length - 15} more line(s)`);
      }
    }
    console.log("");
  }

  console.log("-----------------------------------------------------------------");
  console.log(`Total time: ${(totalMs / 1000).toFixed(1)}s`);

  if (writeSummary) {
    writeGithubSummary(results, totalMs);
  }

  if (failedCount > 0 && !isWarn) {
    console.log(`\n❌ Quality gate failed (${failedCount} check(s) reported issues).`);
    process.exit(1);
  } else {
    console.log(`\n✅ Quality gate passed.`);
    process.exit(0);
  }
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
