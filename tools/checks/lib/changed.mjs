import { execFileSync } from "node:child_process";

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
    return execFileSync("git", args, { cwd, encoding: "utf8", maxBuffer: 1 << 28, stdio: ["ignore", "pipe", "pipe"] })
      .split("\n")
      .filter(Boolean);
  } catch (err) {
    const detail = String(err.stderr ?? err.message).trim();
    throw new Error(`git ${args.join(" ")} failed in ${cwd}: ${detail}`);
  }
}

/**
 * The files a branch changes: the diff against `base` (origin/main unless `BASE_REF` names another), the
 * uncommitted changes and the untracked files. A failing git command throws: a selection built from a missing
 * base would select nothing and pass.
 */
export function listChangedFiles(root, base = process.env.BASE_REF || "origin/main") {
  const committed = listLines(["diff", "--name-only", `${base}...HEAD`], root);
  const worktree = listLines(["diff", "--name-only", "HEAD"], root);
  const untracked = listLines(["ls-files", "--others", "--exclude-standard"], root);
  return [...new Set([...committed, ...worktree, ...untracked])];
}

/** The text of `file` at the merge base of `base` and HEAD; null when it is absent there. */
export function readBaseFile(root, file, base = process.env.BASE_REF || "origin/main") {
  const [fork] = listLines(["merge-base", base, "HEAD"], root);
  try {
    return execFileSync("git", ["show", `${fork}:${file}`], { cwd: root, encoding: "utf8", maxBuffer: 1 << 28, stdio: ["ignore", "pipe", "pipe"] });
  } catch {
    return null;
  }
}

