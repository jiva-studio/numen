import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT } from "./helpers.mjs";

/** The numbers of a Go version such as `1.26.9` or `1.26`; a missing patch is 0. */
function parseGoVersion(version) {
  const [major, minor, patch] = version.split(".").map((part) => Number.parseInt(part, 10) || 0);
  return [major, minor, patch ?? 0];
}

/** Negative when `a` is older than `b`, zero when equal, positive when newer. */
export function compareGoVersions(a, b) {
  const left = parseGoVersion(a);
  const right = parseGoVersion(b);
  for (let i = 0; i < 3; i++) if (left[i] !== right[i]) return left[i] - right[i];
  return 0;
}

/** The Go version `golangci-lint --version` reports it was built with, or undefined. */
export function parseBuiltWithGo(output) {
  return /built with go(\d+\.\d+(?:\.\d+)?)/.exec(output)?.[1];
}

/** The `go` directive of a go.work or go.mod text, or undefined. */
export function parseGoDirective(text) {
  return /^go\s+(\d+\.\d+(?:\.\d+)?)\s*$/m.exec(text)?.[1];
}

/**
 * Why a linter build cannot analyse the repository, or "" when it can: a linter built with a Go older
 * than the workspace stops at the first package it cannot load and reports nothing.
 */
export function getLinterBuildGap(versionOutput, directive) {
  const built = parseBuiltWithGo(versionOutput);
  if (!built) return "the linter does not report the Go version it was built with";
  if (!directive) return "the workspace declares no Go version";
  if (compareGoVersions(built, directive) < 0) return `golangci-lint is built with go${built}, older than go${directive} of go.work`;
  return "";
}

/** Fails when the installed golangci-lint is built with a Go older than the workspace's. */
export function assertGolangciLintBuild() {
  const run = spawnSync("golangci-lint", ["--version"], { cwd: REPO_ROOT, encoding: "utf8" });
  if (run.error || run.status !== 0) {
    throw new Error("golangci-lint is not installed or does not run: " + (run.error?.message ?? run.stderr));
  }
  const directive = parseGoDirective(fs.readFileSync(path.join(REPO_ROOT, "go.work"), "utf8"));
  const gap = getLinterBuildGap(run.stdout, directive);
  if (gap) throw new Error(gap);
}
