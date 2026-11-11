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
  linkInstalledPackages(root, dir);
  const run = (...args) => execFileSync("git", args, { cwd: dir, stdio: "ignore" });
  run("init", "-q");
  fs.appendFileSync(path.join(dir, ".git", "info", "exclude"), "node_modules\n");
  run("add", "-A");
  run("-c", "user.name=checks", "-c", "user.email=checks@invalid", "commit", "-q", "-m", "tree", "--no-verify");
  return dir;
}

/**
 * Links the installed packages of `root` into `dir`, except that a first-party package (a link of
 * `@numen`) points at the same folder of `dir`, so a check follows an import into the reduced tree.
 */
function linkInstalledPackages(root, dir) {
  const modules = path.join(root, "node_modules");
  if (!fs.existsSync(modules)) return;
  const target = path.join(dir, "node_modules");
  fs.mkdirSync(target);
  for (const name of fs.readdirSync(modules)) {
    if (name !== "@numen") {
      fs.symlinkSync(path.join(modules, name), path.join(target, name));
      continue;
    }
    fs.mkdirSync(path.join(target, name));
    for (const own of fs.readdirSync(path.join(modules, name))) {
      const folder = path.relative(root, fs.realpathSync(path.join(modules, name, own)));
      if (fs.existsSync(path.join(dir, folder))) fs.symlinkSync(path.join(dir, folder), path.join(target, name, own));
    }
  }
}
