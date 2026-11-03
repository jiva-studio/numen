import fs from "node:fs";
import path from "node:path";

/**
 * Lists the TypeScript and Vue packages of the repository: every workspace of the root package.json
 * and every plugin that has an ESLint configuration or a frontend, so a package that is added is
 * checked without a change in a check.
 *
 * @param {string} repoRoot
 * @returns {{ relDir: string, name: string }[]}
 */
export function listFrontendPackages(repoRoot) {
  const names = new Set();

  const rootManifest = path.join(repoRoot, "package.json");
  const workspaces = fs.existsSync(rootManifest)
    ? (JSON.parse(fs.readFileSync(rootManifest, "utf8")).workspaces ?? [])
    : [];
  for (const pattern of workspaces) {
    if (pattern.endsWith("/*")) {
      for (const name of listSubdirectories(repoRoot, pattern.slice(0, -2))) names.add(name);
    } else {
      names.add(pattern);
    }
  }

  for (const name of listSubdirectories(repoRoot, "plugins")) {
    if (
      fs.existsSync(path.join(repoRoot, name, "eslint.config.js")) ||
      fs.existsSync(path.join(repoRoot, name, "frontend"))
    ) {
      names.add(name);
    }
  }

  return [...names]
    .filter((name) => fs.existsSync(path.join(repoRoot, name, "eslint.config.js")))
    .map((name) => ({ relDir: name, name }));
}

/**
 * Lists the visible subdirectories of a directory as paths relative to the repository root.
 *
 * @param {string} repoRoot
 * @param {string} relDir
 * @returns {string[]}
 */
function listSubdirectories(repoRoot, relDir) {
  const dir = path.join(repoRoot, relDir);
  if (!fs.existsSync(dir)) return [];
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .filter((e) => e.isDirectory() && !e.name.startsWith(".") && e.name !== "node_modules")
    .map((e) => `${relDir}/${e.name}`);
}
