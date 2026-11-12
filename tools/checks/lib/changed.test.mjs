import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { listChangedFiles } from "./changed.mjs";

const dirs = [];
after(() => dirs.forEach((dir) => fs.rmSync(dir, { recursive: true, force: true })));

function git(dir, ...args) {
  execFileSync("git", ["-c", "user.name=t", "-c", "user.email=t@example.com", ...args], { cwd: dir, stdio: "ignore" });
}

/** A repository whose branch `main` holds `a.txt`, with the work branch checked out. */
function createRepo() {
  const dir = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "changed-")));
  dirs.push(dir);
  git(dir, "init", "-q", "-b", "main");
  fs.writeFileSync(path.join(dir, "a.txt"), "a\n");
  git(dir, "add", ".");
  git(dir, "commit", "-q", "-m", "first");
  git(dir, "checkout", "-q", "-b", "work");
  return dir;
}

describe("listChangedFiles", () => {
  it("lists committed, uncommitted and untracked files once", () => {
    const dir = createRepo();
    fs.writeFileSync(path.join(dir, "b.txt"), "b\n");
    git(dir, "add", ".");
    git(dir, "commit", "-q", "-m", "second");
    fs.writeFileSync(path.join(dir, "a.txt"), "changed\n");
    fs.writeFileSync(path.join(dir, "c.txt"), "c\n");
    assert.deepEqual(listChangedFiles(dir, "main").sort(), ["a.txt", "b.txt", "c.txt"]);
  });

  it("throws when the base does not exist instead of listing nothing", () => {
    const dir = createRepo();
    assert.throws(() => listChangedFiles(dir, "origin/missing"), /git diff --name-only origin\/missing\.\.\.HEAD failed/);
  });

  it("throws outside a repository", () => {
    const dir = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), "changed-none-")));
    dirs.push(dir);
    assert.throws(() => listChangedFiles(dir, "main"), /failed/);
  });
});
