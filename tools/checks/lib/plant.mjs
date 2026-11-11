import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { isMatching } from "./changed.mjs";

/** The tracked files of `root`. */
export function listTrackedFiles(root) {
  return execFileSync("git", ["ls-files", "-z"], { cwd: root, encoding: "utf8", maxBuffer: 1 << 28 }).split("\0").filter(Boolean);
}

/**
 * Copies the tracked files of `root` that match `patterns` into a new git repository and links the
 * installed `node_modules`, so a check can run on a tree that holds nothing else. Returns the folder.
 */
export function plantTree(root, patterns) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "checks-tree-"));
  for (const file of listTrackedFiles(root)) {
    if (!isMatching(file, patterns)) continue;
    const source = path.join(root, file);
    const kind = fs.lstatSync(source, { throwIfNoEntry: false });
    if (!kind) continue;
    fs.mkdirSync(path.dirname(path.join(dir, file)), { recursive: true });
    if (kind.isSymbolicLink()) fs.symlinkSync(fs.readlinkSync(source), path.join(dir, file));
    else if (kind.isFile()) fs.copyFileSync(source, path.join(dir, file));
  }
  const modules = path.join(root, "node_modules");
  if (fs.existsSync(modules)) fs.symlinkSync(modules, path.join(dir, "node_modules"));
  const run = (...args) => execFileSync("git", args, { cwd: dir, stdio: "ignore" });
  run("init", "-q");
  fs.appendFileSync(path.join(dir, ".git", "info", "exclude"), "node_modules\n");
  run("add", "-A");
  run("-c", "user.name=checks", "-c", "user.email=checks@invalid", "commit", "-q", "-m", "tree", "--no-verify");
  return dir;
}
