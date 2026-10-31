import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";
import {
  GO_MODULE_ROOT,
  REPO_ROOT,
  failOnFindings,
  getGoModReferences,
  getGoReferences,
  getImportSpecifiers,
  getVueScript,
  listShippedFiles,
  requireMatches,
} from "./helpers.mjs";

/**
 * The Go libraries and the libraries each one may depend on.
 *
 * A library is a directory under `libs/` that holds a `go.mod`, and it belongs to the family named by
 * its first directory under `libs/`: `libs/plugin-sdk-go/host` is of the family `plugin-sdk-go`. The
 * modules of one family may depend on each other. A dependency between two families is allowed only
 * where the matrix lists it, and a family the matrix does not list fails: a new library states its
 * place before it builds.
 */
export const LIBRARY_DEPENDENCIES = {
  notes: [],
  protocol: [],
  "plugin-catalog": ["notes", "protocol"],
  "plugin-sdk-go": ["notes", "protocol"],
  core: ["notes", "plugin-catalog", "protocol"],
};

/** The family of a library directory such as `libs/plugin-sdk-go/host`. */
export function getFamily(dir) {
  return dir.split("/")[1];
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
  for (const library of libraries) {
    const family = getFamily(library.dir);
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
      const reached = getFamily(target.dir);
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
  const listed = spawnSync("git", ["ls-files", "--", "libs"], { cwd: REPO_ROOT, encoding: "utf8" });
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
    const module = /^\s*module\s+(\S+)/m.exec(modSource)?.[1];
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
  if (libraries.length < 6 || libraries.reduce((total, library) => total + library.references.length, 0) < 30) {
    throw new Error("The library matrix read too little of libs/: a rule checked against nothing");
  }
  failOnFindings("A Go library is outside the library matrix or reaches a library it may not", findLibraryViolations(libraries));
}

/**
 * The TypeScript libraries and the libraries each one may depend on, by package name.
 *
 * A library is a directory under `libs/` that holds a `package.json`. It declares and imports only the
 * first-party packages its row lists, and a library with no row fails, so a new library states its
 * place before it builds.
 */
export const PACKAGE_DEPENDENCIES = {
  "@numen/ui": [],
  "@numen/shell": ["@numen/ui"],
  "@numen/protocol": [],
  "@numen/plugin-sdk": ["@numen/protocol", "@numen/ui"],
  "@numen/plugin-host": ["@numen/plugin-sdk"],
  "@numen/plugin-manager": ["@numen/plugin-sdk", "@numen/ui"],
};

const FIRST_PARTY = /^(@numen\/[^/]+)/;

const DEPENDENCY_FIELDS = ["dependencies", "devDependencies", "peerDependencies", "optionalDependencies"];

/**
 * Lists the findings for packages given as `{ dir, name, declared, imports }`: `declared` holds the
 * first-party package names its `package.json` names, and `imports` the first-party packages its
 * sources import, each as `{ file, line, name }`.
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
    for (const { file, line, name } of one.imports.filter((entry) => entry.name !== one.name && !allowed.includes(entry.name))) {
      found.push({ file, line, symbol: `${one.name} imports ${name}, which the package matrix does not allow` });
    }
  }
  return found;
}

/** Reads every package under `libs/` with the first-party packages it declares and imports. */
export function readPackages() {
  const dirs = listShippedFiles(["libs"])
    .filter((file) => path.basename(file) === "package.json" && !file.includes("node_modules/"))
    .map((file) => path.dirname(file));
  return dirs.map((dir) => {
    const manifest = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, dir, "package.json"), "utf8"));
    const declared = DEPENDENCY_FIELDS.flatMap((field) => Object.keys(manifest[field] ?? {})).filter((name) => FIRST_PARTY.test(name));
    const imports = [];
    for (const file of listShippedFiles([`${dir}/src`]).filter((one) => /\.(ts|vue)$/.test(one))) {
      const text = fs.readFileSync(path.join(REPO_ROOT, file), "utf8");
      const script = file.endsWith(".vue") ? getVueScript(text) : text;
      for (const specifier of getImportSpecifiers(script)) {
        const name = FIRST_PARTY.exec(specifier)?.[1];
        if (name) imports.push({ file, line: script.slice(0, Math.max(0, script.indexOf(specifier))).split("\n").length, name });
      }
    }
    return { dir, name: manifest.name, declared, imports };
  });
}

export function verifyPackageMatrix() {
  const packages = readPackages();
  requireMatches("TypeScript libraries read", packages.length, 5);
  requireMatches("first-party imports of TypeScript libraries", packages.reduce((total, one) => total + one.imports.length, 0), 10);
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
    name: "TypeScript Libraries Depend Only on the Libraries the Matrix Allows (every package.json under libs/ has a row in the package matrix)",
    run: verifyPackageMatrix,
  },
];
