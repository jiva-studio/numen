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

/** A verifier that runs `npm run lint` in the package at `dir`. */
const createLintVerifier = ({ id, scope, module, name, dir }) => ({
  id,
  isLint: true,
  scope,
  module,
  name,
  run: () => {
    exec("npm run lint", path.join(MODULES_DIR, dir));
  },
});

export const FRONTEND_VERIFIERS = [
  {
    id: "all-ts-packages-eslint",
    scope: "libs",
    module: "sdk",
    name: "Every TypeScript Package Has ESLint Configured",
    run: verifyAllTsPackagesHaveEslint,
  },
  createLintVerifier({ id: "ui-lint", scope: "libs", module: "ui", dir: "libs/ui/core",
    name: "UI Library Lint (npm run lint in libs/ui/core; its imports are dependency-graph's)" }),
  createLintVerifier({ id: "shell-lint", scope: "libs", module: "shell", dir: "libs/ui/shell",
    name: "Shell Library Lint (npm run lint in libs/ui/shell; its imports are dependency-graph's)" }),
  createLintVerifier({ id: "plugin-host-lint", scope: "libs", module: "host", dir: "libs/plugin-host-ts",
    name: "Plugin Host Lint (npm run lint in libs/plugin-host-ts; its imports are dependency-graph's)" }),
  createLintVerifier({ id: "desktop-editor-fsd", scope: "apps", module: "desktop", dir: "apps/desktop/editor",
    name: "Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> pages -> app)" }),
  createLintVerifier({ id: "desktop-flashcards-fsd", scope: "apps", module: "desktop", dir: "apps/desktop/flashcards",
    name: "Desktop Flashcards FSD Layer Hierarchy" }),
];
