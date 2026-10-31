import fs from "node:fs";
import path from "node:path";

/**
 * Lists the Go modules of the repository: every `use` entry of go.work and every directory of
 * plugins/ holding a go.mod, so a module or a plugin that is added is checked without a change in a
 * check.
 *
 * @param {string} repoRoot
 * @returns {{ name: string, dir: string }[]}
 */
export function listGoModules(repoRoot) {
  const names = new Set();

  const workFile = path.join(repoRoot, "go.work");
  if (fs.existsSync(workFile)) {
    const work = fs.readFileSync(workFile, "utf8");
    for (const match of work.matchAll(/^\s*(?:use\s+)?(\.\/[^\s)]+)\s*$/gm)) {
      names.add(path.posix.normalize(match[1]));
    }
  }

  for (const name of listSubdirectories(repoRoot, "plugins")) names.add(name);

  return [...names]
    .map((name) => ({ name, dir: path.join(repoRoot, name) }))
    .filter((m) => fs.existsSync(path.join(m.dir, "go.mod")));
}

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

/**
 * Finds the Go lint configuration a module is checked with: its own copy when it carries one, the
 * nearest copy in a parent directory otherwise, the root copy at last.
 *
 * @param {string} repoRoot
 * @param {string} moduleDir
 * @param {string} fileName `.golangci.yml` or `.golangci-structure.yml`
 * @returns {string}
 */
export function findGoLintConfig(repoRoot, moduleDir, fileName) {
  let dir = moduleDir;
  while (dir.startsWith(repoRoot)) {
    const candidate = path.join(dir, fileName);
    if (fs.existsSync(candidate)) return candidate;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return path.join(repoRoot, fileName);
}
