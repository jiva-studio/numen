import fs from "node:fs";
import path from "node:path";
import { REPO_ROOT, failOnFindings, listTrackedFiles, parsePackageJson, requireMatches } from "../../../lib/graph.mjs";

/** The folders of the TypeScript packages that can be moved to a repository of their own and published. */
export const SHARED_DIR = "libs/shared";
export const EXTRACTABLE_DIRS = ["libs/plugin/sdk-ts", "libs/ui", SHARED_DIR];

/** The folder whose packages the extractable set serves: they may require only the extractable set. */
export const PLUGINS_DIR = "plugins";

const DEPENDENCY_FIELDS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"];

/** Installed packages, build output, generated code and fixtures, which are not packages of the repository. */
const NOT_SOURCE = /(^|\/)(node_modules|dist|gen|testdata)\//;

const isWithin = (file, dir) => file === dir || file.startsWith(dir + "/");

/** The tracked package.json files as `{ file, manifest }`. */
function readManifests() {
  return listTrackedFiles()
    .filter((file) => path.basename(file) === "package.json" && !NOT_SOURCE.test(file))
    .map((file) => ({ file, manifest: parsePackageJson(fs.readFileSync(path.join(REPO_ROOT, file), "utf8")) }));
}

/**
 * Lists what the packages of `manifests` (`{ file, manifest }`) in `subjectDirs` declare, in any dependency
 * field, among the packages of the repository that lie outside `allowedDirs`. A dependency that no package of
 * the repository carries the name of is third-party code and passes.
 */
export function findRequirementsOutside(manifests, subjectDirs, allowedDirs, reason) {
  const folderOf = new Map(manifests.map(({ file, manifest }) => [manifest.name, path.posix.dirname(file)]));
  const found = [];
  for (const { file, manifest } of manifests) {
    if (!subjectDirs.some((dir) => isWithin(file, dir))) continue;
    for (const field of DEPENDENCY_FIELDS) {
      for (const name of Object.keys(manifest[field] ?? {})) {
        const folder = folderOf.get(name);
        if (folder === undefined || allowedDirs.some((dir) => isWithin(folder, dir))) continue;
        found.push({ file, line: 1, symbol: `${manifest.name} declares ${name} in ${field} (${folder}): ${reason}` });
      }
    }
  }
  return found;
}

export function findSharedLeafViolations(manifests) {
  return findRequirementsOutside(manifests, [SHARED_DIR], [SHARED_DIR], "a package in libs/shared is a leaf: it requires only third-party packages and other libs/shared packages");
}

export function findExtractableViolations(manifests) {
  return findRequirementsOutside(
    manifests,
    [...EXTRACTABLE_DIRS, PLUGINS_DIR],
    EXTRACTABLE_DIRS,
    "an extractable package requires only third-party packages, the plugin SDK, @numen/ui, @numen/workbench and libs/shared",
  );
}

/** A package under libs/shared requires only third-party packages and other libs/shared packages. */
export function verifySharedLibrariesAreLeaves() {
  const manifests = readManifests();
  requireMatches("packages read for the extractable set", manifests.length);
  failOnFindings("A TypeScript package in libs/shared requires a package outside libs/shared", findSharedLeafViolations(manifests));
}

/** A package of the plugin SDK, of libs/ui and every plugin requires only third-party packages and the extractable set. */
export function verifyExtractableSetRequiresOnlyItself() {
  const manifests = readManifests();
  requireMatches(
    "packages of the extractable set and plugins",
    manifests.filter(({ file }) => [...EXTRACTABLE_DIRS, PLUGINS_DIR].some((dir) => isWithin(file, dir))).length,
  );
  failOnFindings("A package of the extractable set or a plugin requires a package outside the set", findExtractableViolations(manifests));
}

export const EXTRACTABLE_SET_VERIFIERS = [
  {
    id: "shared-libraries-are-leaves",
    scope: "libs",
    module: "extractable",
    name: "Shared Libraries Are Leaves (a package under libs/shared declares no package of the repository outside libs/shared; package.json)",
    run: verifySharedLibrariesAreLeaves,
  },
  {
    id: "extractable-set-requires-only-itself",
    scope: "libs",
    module: "extractable",
    name: "Extractable Set Requires Only Itself (the plugin SDK, libs/ui and every plugin declare no package of the repository outside the plugin SDK, libs/ui and libs/shared; package.json)",
    run: verifyExtractableSetRequiresOnlyItself,
  },
];
