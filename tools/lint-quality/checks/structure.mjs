import fs from "node:fs";
import path from "node:path";

const MAX_SOURCE_FILES_PER_DIR = 20;

const IGNORED_DIRS = new Set([
  "node_modules",
  ".git",
  ".claude",
  "dist",
  "built",
  "coverage",
  "gen",
  "proto",
  "assets",
  "fixtures",
  "sql",
  ".storybook",
]);

/**
 * Checks if a file is an authored source code file (excluding generated files).
 *
 * @param {string} filename
 * @returns {boolean}
 */
function isSourceFile(filename) {
  if (
    filename.endsWith(".pb.go") ||
    filename.endsWith(".connect.go") ||
    filename.endsWith("_pb.ts") ||
    filename.endsWith("_test.go") ||
    filename.endsWith(".spec.ts") ||
    filename.endsWith(".test.ts") ||
    filename.endsWith(".spec.js") ||
    filename.endsWith(".test.js")
  ) {
    return false;
  }
  return /\.(go|ts|vue)$/.test(filename);
}

/**
 * Recursively scans directory and counts authored source files in each folder.
 *
 * @param {string} dir
 * @param {string} repoRoot
 * @returns {Array<{ dir: string, relDir: string, count: number, files: string[] }>}
 */
function scanDirectory(dir, repoRoot) {
  const results = [];

  let entries;
  try {
    entries = fs.readdirSync(dir, { withFileTypes: true });
  } catch {
    return results;
  }

  const sourceFiles = [];
  const subdirs = [];

  for (const entry of entries) {
    if (entry.isFile() && isSourceFile(entry.name)) {
      sourceFiles.push(entry.name);
    } else if (entry.isDirectory() && !IGNORED_DIRS.has(entry.name)) {
      subdirs.push(path.join(dir, entry.name));
    }
  }

  const relDir = path.relative(repoRoot, dir);
  if (sourceFiles.length > MAX_SOURCE_FILES_PER_DIR) {
    results.push({
      dir,
      relDir,
      count: sourceFiles.length,
      files: sourceFiles,
    });
  }

  for (const subdir of subdirs) {
    results.push(...scanDirectory(subdir, repoRoot));
  }

  return results;
}

/**
 * Validates directory structure to ensure directories do not become monolithic flat file dumps.
 *
 * @param {{ repoRoot: string, warnOnly?: boolean, targetDirs?: string[] }} options
 * @returns {Promise<{ id: string, name: string, pass: boolean, summary: string, details?: string, durationMs: number }>}
 */
export async function runStructureCheck({
  repoRoot,
  warnOnly = false,
  targetDirs = null,
  changedFiles = null,
  changedOnly = false,
}) {
  const start = Date.now();
  const rootDirs = ["libs", "plugins", "apps"].map((d) => path.join(repoRoot, d));

  const allOvercrowded = rootDirs.flatMap((d) => scanDirectory(d, repoRoot));

  let activeTargetDirs = targetDirs;
  if (changedOnly && changedFiles && changedFiles.length > 0 && !activeTargetDirs) {
    activeTargetDirs = [...new Set(changedFiles.map((f) => path.dirname(f)))];
  }

  let evaluated = allOvercrowded;
  if (activeTargetDirs && activeTargetDirs.length > 0) {
    evaluated = allOvercrowded.filter((item) =>
      activeTargetDirs.some((t) => item.relDir === t || item.relDir.startsWith(t + "/"))
    );
  }

  evaluated.sort((a, b) => b.count - a.count);

  const durationMs = Date.now() - start;
  const pass = evaluated.length === 0 || warnOnly;

  const summary =
    evaluated.length === 0
      ? `Zero directories exceed max file limit (<= ${MAX_SOURCE_FILES_PER_DIR} source files per folder)`
      : `${evaluated.length} directories exceed max file limit (> ${MAX_SOURCE_FILES_PER_DIR} source files per folder)`;

  const details =
    evaluated.length > 0
      ? evaluated
          .map(
            (item) =>
              `✖ ${item.relDir}: ${item.count} source files (limit: ${MAX_SOURCE_FILES_PER_DIR})`
          )
          .join("\n")
      : undefined;

  return {
    id: "structure",
    name: "Directory Structure & Size (max 20 source files)",
    pass,
    summary,
    details,
    durationMs,
  };
}
