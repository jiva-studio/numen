import { execFileSync } from "node:child_process";
import path from "node:path";

/** The regular expression of a path pattern: `**` crosses folders, `*` and `?` stay in one, `{a,b}` is a choice. */
export function getPathPattern(pattern) {
  let source = "";
  for (let i = 0; i < pattern.length; i++) {
    const c = pattern[i];
    if (c === "*" && pattern[i + 1] === "*") {
      const slash = pattern[i + 2] === "/";
      source += slash ? "(?:.*/)?" : ".*";
      i += slash ? 2 : 1;
    } else if (c === "*") source += "[^/]*";
    else if (c === "?") source += "[^/]";
    else if (c === "{") {
      const end = pattern.indexOf("}", i);
      source += "(?:" + pattern.slice(i + 1, end).split(",").map(escape).join("|") + ")";
      i = end;
    } else source += escape(c);
  }
  return new RegExp("^" + source + "$");
}

function escape(text) {
  return text.replace(/[.+^$()|[\]\\]/g, "\\$&");
}

/** Whether `file` matches one of `patterns`. */
export function isMatching(file, patterns) {
  return patterns.some((pattern) => getPathPattern(pattern).test(file));
}

function listLines(args, cwd) {
  try {
    return execFileSync("git", args, { cwd, encoding: "utf8", maxBuffer: 1 << 28, stdio: ["ignore", "pipe", "ignore"] })
      .split("\n")
      .filter(Boolean);
  } catch {
    return [];
  }
}

/**
 * The files a branch changes: the diff against `base` (origin/main unless `BASE_REF` names another), the
 * uncommitted changes and the untracked files.
 */
export function listChangedFiles(root, base = process.env.BASE_REF || "origin/main") {
  const committed = listLines(["diff", "--name-only", `${base}...HEAD`], root);
  const worktree = listLines(["diff", "--name-only", "HEAD"], root);
  const untracked = listLines(["ls-files", "--others", "--exclude-standard"], root);
  return [...new Set([...committed, ...worktree, ...untracked])];
}

/** The checks directory relative to the repository, so a change in the runner or the library selects everything. */
export function getCheckerPaths(root, checksDir) {
  const relative = path.relative(root, checksDir).split(path.sep).join("/");
  return { runner: [`${relative}/run.mjs`, `${relative}/rules.json`, `${relative}/lib/*.mjs`, `${relative}/structure/go/load/*.go`], dir: relative };
}
