import assert from "node:assert/strict";
import { test } from "node:test";
import { PACKAGE_DEPENDENCIES, findLibraryViolations, findPackageViolations, readLibraries, readPackages } from "./libraries.mjs";

const ROOT = "github.com/jiva-studio/numen/";

const library = (dir, references = []) => ({
  dir,
  module: ROOT + dir,
  references: references.map((value, index) => ({ file: `${dir}/a.go`, line: index + 1, value: ROOT + value })),
});

const MATRIX = { notes: [], protocol: [], catalog: ["notes", "protocol"], sdk: ["protocol"] };

test("a library that reaches a family its row lists is clean", () => {
  const libraries = [
    library("libs/notes"),
    library("libs/protocol"),
    library("libs/catalog", ["libs/notes/domain", "libs/protocol/gen/numen/v1"]),
  ];
  assert.deepEqual(findLibraryViolations(libraries, MATRIX), []);
});

test("a library that reaches a family its row omits is reported with the reference", () => {
  const libraries = [library("libs/catalog"), library("libs/sdk", ["libs/catalog/manifest"])];
  const found = findLibraryViolations(libraries, MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /family sdk reaches catalog/);
  assert.equal(found[0].file, "libs/sdk/a.go");
});

test("the modules of one family may depend on each other", () => {
  const libraries = [library("libs/sdk"), library("libs/sdk/host", ["libs/sdk/cache"]), library("libs/sdk/cache", ["libs/sdk"])];
  assert.deepEqual(findLibraryViolations(libraries, MATRIX), []);
});

test("a library whose family the matrix does not list is reported", () => {
  const found = findLibraryViolations([library("libs/unclassified")], MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /no family "unclassified"/);
});

test("a library that reaches an application or a plugin is reported", () => {
  const libraries = [library("libs/catalog", ["apps/desktop/internal/platform", "plugins/readers-pdf"])];
  const found = findLibraryViolations(libraries, MATRIX);
  assert.equal(found.length, 2);
  assert.match(found[0].symbol, /outside libs/);
});

test("the libraries of this repository are read, each with a module path", () => {
  const libraries = readLibraries();
  assert.ok(libraries.some((one) => one.dir === "libs/plugin-catalog"));
  assert.ok(libraries.every((one) => one.module));
});

const PACKAGE_MATRIX = { "@numen/base": [], "@numen/mid": ["@numen/base"] };

const pkg = (name, declared = [], imports = []) => ({
  dir: `libs/${name}`,
  name: `@numen/${name}`,
  declared: declared.map((one) => `@numen/${one}`),
  imports: imports.map((one, index) => ({ file: `libs/${name}/src/a.ts`, line: index + 1, name: `@numen/${one}` })),
});

test("a package that declares and imports the packages its row lists is clean", () => {
  const packages = [pkg("base"), pkg("mid", ["base"], ["base", "mid"])];
  assert.deepEqual(findPackageViolations(packages, PACKAGE_MATRIX), []);
});

test("a package that declares a package its row omits is reported", () => {
  const found = findPackageViolations([pkg("base", ["mid"])], PACKAGE_MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /@numen\/base declares @numen\/mid/);
  assert.equal(found[0].file, "libs/base/package.json");
});

test("a package that imports a package its row omits is reported with the line", () => {
  const found = findPackageViolations([pkg("base", [], ["base", "mid"])], PACKAGE_MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /@numen\/base imports @numen\/mid/);
  assert.equal(found[0].line, 2);
});

test("a package with no row is reported", () => {
  const found = findPackageViolations([pkg("unlisted")], PACKAGE_MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /no row in the package matrix/);
});

test("the packages of this repository are read, the plugin manager among them", () => {
  const packages = readPackages();
  const manager = packages.find((one) => one.name === "@numen/plugin-manager");
  assert.deepEqual([...manager.declared].sort(), ["@numen/plugin-sdk", "@numen/ui"]);
  assert.ok(manager.imports.length > 0);
  assert.deepEqual(findPackageViolations(packages, PACKAGE_DEPENDENCIES), []);
});
