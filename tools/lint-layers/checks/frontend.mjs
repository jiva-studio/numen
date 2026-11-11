import { runTool } from "../../shared/tools.mjs";
import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, requireMatches } from "./helpers.mjs";

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
  requireMatches("TypeScript packages read", dirs.length);
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
];
