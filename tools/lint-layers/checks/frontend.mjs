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
  // 12 on the tree when the floor was set; the floor is that count less 5%.
  requireMatches("TypeScript packages read", dirs.length, 11);
  failOnFindings("TypeScript package missing eslint.config.js", findTsPackagesWithoutEslint(dirs));
}

/** A verifier that runs `npm run lint` in the package at `dir`. */
const createLintVerifier = (id, scope, module, dir, name) => ({
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
  createLintVerifier("ui-components-lint", "libs", "ui", "libs/ui/components",
    "UI Library Lint (npm run lint in libs/ui/components; its imports are dependency-graph's)"),
  createLintVerifier("workbench-lint", "libs", "shell", "libs/ui/workbench",
    "Workbench Library Lint (npm run lint in libs/ui/workbench; its imports are dependency-graph's)"),
  createLintVerifier("plugin-loader-lint", "libs", "host", "libs/plugin/loader",
    "Plugin Loader Lint (npm run lint in libs/plugin/loader; its imports are dependency-graph's)"),
  createLintVerifier("desktop-editor-fsd", "apps", "desktop", "apps/desktop/editor",
    "Desktop Editor FSD Layer Hierarchy (shared -> entities -> features -> pages -> app)"),
  createLintVerifier("desktop-flashcards-fsd", "apps", "desktop", "apps/desktop/flashcards",
    "Desktop Flashcards FSD Layer Hierarchy"),
  createLintVerifier("mobile-fsd", "apps", "mobile", "apps/mobile",
    "Mobile FSD Layer Hierarchy"),
];
