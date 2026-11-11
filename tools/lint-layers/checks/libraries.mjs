import { runTool } from "../../shared/tools.mjs";
import fs from "node:fs";
import path from "node:path";
import {
  GO_MODULE_ROOT,
  REPO_ROOT,
  SCRIPT_DIR,
  failOnFindings,
  getGoModReferences,
  getGoReferences,
  listShippedFiles,
  parseGoMod,
  requireMatches,
} from "./helpers.mjs";

/** The path of the matrix both the library check and the import graph read. */
export const LIBRARY_MATRIX_FILE = path.join(SCRIPT_DIR, "library-matrix.json");

/** Reads the matrix: the Go families and the TypeScript packages, each with the ones it may depend on. */
export function readLibraryMatrix(file = LIBRARY_MATRIX_FILE) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

/**
 * The Go libraries and the libraries each one may depend on, from `library-matrix.json`.
 *
 * A library is a directory under `libs/` that holds a `go.mod`. Its family is the path under `libs/` of the
 * outermost library that contains it: `libs/plugin/sdk-go/host` is of the family `plugin/sdk-go` because
 * `libs/plugin/sdk-go` holds a `go.mod` too, and `libs/note` is of the family `note`. The
 * modules of one family may depend on each other. A dependency between two families is allowed only
 * where the matrix lists it, and a family the matrix does not list fails: a new library states its
 * place before it builds.
 */
export const LIBRARY_DEPENDENCIES = readLibraryMatrix().go;

/** The family of the library at `dir`, read from where the `go.mod` files are: `dirs` holds every library directory. */
export function getFamily(dir, dirs) {
  const outermost = dirs
    .filter((one) => dir === one || dir.startsWith(one + "/"))
    .sort((a, b) => a.length - b.length)[0];
  return (outermost ?? dir).replace(/^libs\//, "");
}

/**
 * Lists the findings for libraries given as `{ dir, module, references }`, where each reference is
 * `{ file, line, value }` for a path under the repository's Go module root. Every library must have a
 * family in the matrix, and every reference must reach the same family, a family the matrix allows, or
 * nothing outside `libs/`.
 */
export function findLibraryViolations(libraries, matrix = LIBRARY_DEPENDENCIES) {
  const found = [];
  const modules = libraries.map((library) => ({ dir: library.dir, path: library.module }));
  const dirs = libraries.map((library) => library.dir);
  for (const library of libraries) {
    const family = getFamily(library.dir, dirs);
    const allowed = matrix[family];
    if (!allowed) {
      found.push({
        file: `${library.dir}/go.mod`,
        line: 1,
        symbol: `library ${library.dir} has no family "${family}" in the library matrix`,
      });
      continue;
    }
    for (const reference of library.references) {
      const target = modules.find((one) => reference.value === one.path || reference.value.startsWith(one.path + "/"));
      if (!target) {
        if (!reference.value.startsWith(GO_MODULE_ROOT + "libs/")) {
          found.push({
            file: reference.file,
            line: reference.line,
            symbol: `a library reaches outside libs/: ${reference.value}`,
          });
        }
        continue;
      }
      const reached = getFamily(target.dir, dirs);
      if (reached !== family && !allowed.includes(reached)) {
        found.push({
          file: reference.file,
          line: reference.line,
          symbol: `library family ${family} reaches ${reached}, which the library matrix does not allow: ${reference.value}`,
        });
      }
    }
  }
  return found;
}

/** Lists the tracked files under `libs/` that the library check reads. */
function listLibraryFiles() {
  const listed = runTool("git", ["ls-files", "--", "libs"], { cwd: REPO_ROOT, encoding: "utf8" });
  return (listed.stdout || "")
    .split("\n")
    .filter((file) => file && !file.includes("node_modules/") && fs.existsSync(path.join(REPO_ROOT, file)));
}

/** Reads every Go library under `libs/` with the references its `go.mod` and its Go files make to repository modules. */
export function readLibraries() {
  const files = listLibraryFiles();
  const dirs = files.filter((file) => path.basename(file) === "go.mod").map((file) => path.dirname(file));
  const libraries = dirs.map((dir) => {
    const modFile = `${dir}/go.mod`;
    const modSource = fs.readFileSync(path.join(REPO_ROOT, modFile), "utf8");
    const module = parseGoMod(modSource).Module?.Path;
    return { dir, module, references: [] };
  });
  const ownerOf = (file) =>
    libraries
      .filter((library) => file.startsWith(library.dir + "/"))
      .sort((a, b) => b.dir.length - a.dir.length)[0];
  for (const library of libraries) {
    const modFile = `${library.dir}/go.mod`;
    const modSource = fs.readFileSync(path.join(REPO_ROOT, modFile), "utf8");
    for (const { line, value } of getGoModReferences(modSource).imports) {
      library.references.push({ file: modFile, line, value });
    }
  }
  for (const file of files.filter((one) => one.endsWith(".go"))) {
    const owner = ownerOf(file);
    if (!owner) continue;
    const source = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
    for (const { line, value } of getGoReferences(source).imports) {
      if (value.startsWith(GO_MODULE_ROOT)) owner.references.push({ file, line, value });
    }
  }
  return libraries;
}

export function verifyLibraryMatrix() {
  const libraries = readLibraries();
  requireMatches("Go libraries read", libraries.length);
  requireMatches("Go references of libraries", libraries.reduce((total, library) => total + library.references.length, 0));
  failOnFindings("A Go library is outside the library matrix or reaches a library it may not", findLibraryViolations(libraries));
}

/**
 * The TypeScript libraries and the libraries each one may depend on, by package name.
 *
 * A library is a directory under `libs/` that holds a `package.json`. It declares only the first-party
 * packages its row lists, and a library with no row fails, so a new library states its place before it
 * builds. `dependency-graph` holds its imports to the same row.
 */
export const PACKAGE_DEPENDENCIES = readLibraryMatrix().packages;

const FIRST_PARTY = /^(@numen\/[^/]+)/;

const DEPENDENCY_FIELDS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"];

/**
 * Lists the findings for packages given as `{ dir, name, declared }`: `declared` holds the first-party
 * package names its `package.json` names.
 */
export function findPackageViolations(packages, matrix = PACKAGE_DEPENDENCIES) {
  const found = [];
  for (const one of packages) {
    const allowed = matrix[one.name];
    if (!allowed) {
      found.push({ file: `${one.dir}/package.json`, line: 1, symbol: `package ${one.name} has no row in the package matrix` });
      continue;
    }
    for (const name of one.declared.filter((declared) => declared !== one.name && !allowed.includes(declared))) {
      found.push({ file: `${one.dir}/package.json`, line: 1, symbol: `${one.name} declares ${name}, which the package matrix does not allow` });
    }
  }
  return found;
}

/** Reads every package under `libs/` with the first-party packages it declares. */
export function readPackages() {
  const dirs = listShippedFiles(["libs"])
    .filter((file) => path.basename(file) === "package.json" && !file.includes("node_modules/"))
    .map((file) => path.dirname(file));
  return dirs.map((dir) => {
    const manifest = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, dir, "package.json"), "utf8"));
    const declared = DEPENDENCY_FIELDS.flatMap((field) => Object.keys(manifest[field] ?? {})).filter((name) => FIRST_PARTY.test(name));
    return { dir, name: manifest.name, declared };
  });
}

export function verifyPackageMatrix() {
  const packages = readPackages();
  requireMatches("TypeScript libraries read", packages.length);
  requireMatches("first-party declarations of TypeScript libraries", packages.reduce((total, one) => total + one.declared.length, 0));
  failOnFindings("A TypeScript library is outside the package matrix or reaches a library it may not", findPackageViolations(packages));
}

export const LIBRARY_VERIFIERS = [
  {
    id: "library-matrix",
    scope: "libs",
    module: "libraries",
    name: "Go Libraries Depend Only on the Libraries the Matrix Allows (every go.mod under libs/ has a family in the matrix)",
    run: verifyLibraryMatrix,
  },
  {
    id: "package-matrix",
    scope: "libs",
    module: "libraries",
    name: "TypeScript Libraries Declare Only the Libraries the Matrix Allows (every package.json under libs/ has a row in the package matrix)",
    run: verifyPackageMatrix,
  },
];
