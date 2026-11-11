import assert from "node:assert/strict";
import fs from "node:fs";
import { test } from "node:test";
import { LIBRARY_DEPENDENCIES, PACKAGE_DEPENDENCIES, findLibraryViolations, findPackageViolations, getFamily, readLibraries, readLibraryMatrix, readPackages, LIBRARY_MATRIX_FILE } from "./check.mjs";
import { REPO_ROOT } from "../../../lib/graph.mjs";
import { createCruiseConfig } from "../dependency-graph/check.mjs";

const ROOT = "github.com/jiva-studio/numen/";

const library = (dir, references = []) => ({
  dir,
  module: ROOT + dir,
  references: references.map((value, index) => ({ file: `${dir}/a.go`, line: index + 1, value: ROOT + value })),
});

const MATRIX = { "libs/note": [], "libs/core/protocol": [], "libs/catalog": ["libs/note", "libs/core/protocol"], "libs/sdk": ["libs/core/protocol"] };

test("a library that reaches a family its row lists is clean", () => {
  const libraries = [
    library("libs/note"),
    library("libs/core/protocol"),
    library("libs/catalog", ["libs/note/domain", "libs/core/protocol/gen/numen/v1"]),
  ];
  assert.deepEqual(findLibraryViolations(libraries, MATRIX), []);
});

test("a library that reaches a family its row omits is reported with the reference", () => {
  const libraries = [library("libs/catalog"), library("libs/sdk", ["libs/catalog/manifest"])];
  const found = findLibraryViolations(libraries, MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /family libs\/sdk reaches libs\/catalog/);
  assert.equal(found[0].file, "libs/sdk/a.go");
});

test("the modules of one family may depend on each other", () => {
  const libraries = [library("libs/sdk"), library("libs/sdk/host", ["libs/sdk/cache"]), library("libs/sdk/cache", ["libs/sdk"])];
  assert.deepEqual(findLibraryViolations(libraries, MATRIX), []);
});

test("a library whose family the matrix does not list is reported", () => {
  const found = findLibraryViolations([library("libs/unclassified")], MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /no family "libs\/unclassified"/);
});

test("a library that reaches an application or a plugin is reported", () => {
  const libraries = [library("libs/catalog", ["apps/desktop/internal/platform", "plugins/readers-pdf"])];
  const found = findLibraryViolations(libraries, MATRIX);
  assert.equal(found.length, 2);
  assert.match(found[0].symbol, /outside libs/);
});

test("the plugin SDK family may reach the notes library and the protocol", () => {
  assert.deepEqual(LIBRARY_DEPENDENCIES["libs/plugin/sdk-go"], ["libs/note", "libs/core/protocol"]);
  const libraries = [
    library("libs/note"),
    library("libs/core/protocol"),
    library("libs/plugin/sdk-go"),
    library("libs/plugin/sdk-go/host", ["libs/note", "libs/note/domain", "libs/core/protocol/gen/numen/v1"]),
  ];
  assert.deepEqual(findLibraryViolations(libraries, LIBRARY_DEPENDENCIES), []);
});

test("the plugin SDK family is not allowed to reach the core", () => {
  const libraries = [library("libs/core/kernel"), library("libs/plugin/sdk-go", ["libs/core/kernel/port"])];
  const found = findLibraryViolations(libraries, LIBRARY_DEPENDENCIES);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /family libs\/plugin\/sdk-go reaches libs\/core\/kernel/);
});

test("the libraries of this repository are read, each with a module path", () => {
  const libraries = readLibraries();
  assert.ok(libraries.some((one) => one.dir === "libs/plugin/catalog"));
  assert.ok(libraries.every((one) => one.module));
});

const PACKAGE_MATRIX = { "@numen/base": [], "@numen/mid": ["@numen/base"] };

const pkg = (name, declared = []) => ({
  dir: `libs/${name}`,
  name: `@numen/${name}`,
  declared: declared.map((one) => `@numen/${one}`),
});

test("a package that declares the packages its row lists is clean", () => {
  assert.deepEqual(findPackageViolations([pkg("base"), pkg("mid", ["base"])], PACKAGE_MATRIX), []);
});

test("a package that declares a package its row omits is reported", () => {
  const found = findPackageViolations([pkg("base", ["mid"])], PACKAGE_MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /@numen\/base declares @numen\/mid/);
  assert.equal(found[0].file, "libs/base/package.json");
});

test("a package with no row is reported", () => {
  const found = findPackageViolations([pkg("unlisted")], PACKAGE_MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /no row in the package matrix/);
});

test("the packages of this repository are read, the plugin manager among them", () => {
  const packages = readPackages();
  const manager = packages.find((one) => one.name === "@numen/plugin-manager");
  assert.deepEqual([...manager.declared].sort(), ["@numen/ui"]);
  assert.deepEqual(findPackageViolations(packages, PACKAGE_DEPENDENCIES), []);
});

test("the rows allow the plugin manager the UI library, the plugin host the SDK and the core client, the SDK the schema and the UI build helpers, and the core client the schema", () => {
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-manager"], ["@numen/ui"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-loader"], ["@numen/plugin-sdk", "@numen/core-client"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-sdk"], ["@numen/protocol", "@numen/ui"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/core-client"], ["@numen/protocol"]);
});

test("a plugin host that declares the schema is reported", () => {
  const host = { dir: "libs/plugin/loader", name: "@numen/plugin-loader", declared: ["@numen/plugin-sdk", "@numen/core-client", "@numen/protocol"] };
  const found = findPackageViolations([host], PACKAGE_DEPENDENCIES);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /declares @numen\/protocol/);
});

test("the family of a library is the path of its outermost library, read from where the go.mod files are", () => {
  const dirs = ["libs/note", "libs/plugin/sdk-go", "libs/plugin/sdk-go/host", "libs/core/kernel"];
  assert.equal(getFamily("libs/note", dirs), "libs/note");
  assert.equal(getFamily("libs/plugin/sdk-go/host", dirs), "libs/plugin/sdk-go");
  assert.equal(getFamily("libs/core/kernel", dirs), "libs/core/kernel");
});

test("a nested library with no outer library is its own family, and a library with no row is reported", () => {
  const libraries = [library("libs/group/inner")];
  const found = findLibraryViolations(libraries, MATRIX);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /no family "libs\/group\/inner"/);
});

test("the import graph and the library check read the same matrix file", () => {
  const matrix = readLibraryMatrix();
  assert.deepEqual(matrix.go, LIBRARY_DEPENDENCIES);
  assert.deepEqual(matrix.packages, PACKAGE_DEPENDENCIES);
  const rows = createCruiseConfig(REPO_ROOT, matrix.packages).forbidden.map((rule) => rule.name);
  assert.ok(rows.some((name) => name.endsWith("-reaches-only-its-packages")));
});

test("the matrix file is keyed by the directory of each package, and names the package that directory holds", () => {
  const rows = JSON.parse(fs.readFileSync(LIBRARY_MATRIX_FILE, "utf8")).packages;
  for (const one of readPackages()) assert.equal(rows[one.dir]?.name, one.name, `${one.dir} is keyed by its directory`);
});
