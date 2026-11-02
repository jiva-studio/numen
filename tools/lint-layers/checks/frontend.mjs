import { runTool } from "../../shared/tools.mjs";
import fs from "node:fs";
import path from "node:path";
import { MODULES_DIR, REPO_ROOT, exec, failOnFindings, requireMatches } from "./helpers.mjs";

/** The TypeScript packages of `libs` and `apps`: each directory that holds a `tsconfig.json` and a `package.json`. */
function listTsPackages() {
  const listed = runTool("git", ["ls-files", "--", "libs/*tsconfig.json", "apps/*tsconfig.json"], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => path.basename(file) === "tsconfig.json")
    .map((file) => path.dirname(file))
    .filter((dir) => fs.existsSync(path.join(REPO_ROOT, dir, "package.json")));
}

/** The packages of `dirs` that have no `eslint.config.js`. */
export function findTsPackagesWithoutEslint(dirs, hasConfig = (dir) => fs.existsSync(path.join(REPO_ROOT, dir, "eslint.config.js"))) {
  return dirs.filter((dir) => !hasConfig(dir)).map((dir) => ({ file: `${dir}/tsconfig.json`, line: 1, symbol: "missing eslint.config.js" }));
}

export function verifyAllTsPackagesHaveEslint() {
  const dirs = listTsPackages();
  requireMatches("TypeScript packages read", dirs.length, 8);
  failOnFindings("TypeScript package missing eslint.config.js", findTsPackagesWithoutEslint(dirs));
}

export const FRONTEND_VERIFIERS = [
  {
    id: "all-ts-packages-eslint",
    scope: "libs",
    module: "sdk",
    name: "Every TypeScript Package Has ESLint Configured",
    run: verifyAllTsPackagesHaveEslint,
  },
  {
    id: "ui-lint",
    isLint: true,
    scope: "libs",
    module: "ui",
    name: "UI Library Lint (npm run lint in libs/ui/core; its imports are dependency-graph's)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "libs/ui/core"));
    },
  },
  {
    id: "shell-lint",
    isLint: true,
    scope: "libs",
    module: "shell",
    name: "Shell Library Lint (npm run lint in libs/ui/shell; its imports are dependency-graph's)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "libs/ui/shell"));
    },
  },
  {
    id: "plugin-host-lint",
    isLint: true,
    scope: "libs",
    module: "host",
    name: "Plugin Host Lint (npm run lint in libs/plugin-host-ts; its imports are dependency-graph's)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "libs/plugin-host-ts"));
    },
  },
  {
    id: "desktop-editor-fsd",
    isLint: true,
    scope: "apps",
    module: "desktop",
    name: "Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> pages -> app)",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/editor"));
    },
  },
  {
    id: "desktop-flashcards-fsd",
    isLint: true,
    scope: "apps",
    module: "desktop",
    name: "Desktop Flashcards FSD Layer Hierarchy",
    run: () => {
      exec("npm run lint", path.join(MODULES_DIR, "apps/desktop/flashcards"));
    },
  },
];
