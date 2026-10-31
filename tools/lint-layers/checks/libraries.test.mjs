import assert from "node:assert/strict";
import { test } from "node:test";
import { LIBRARY_DEPENDENCIES, PACKAGE_DEPENDENCIES, findLibraryViolations, findPackageViolations, readLibraries, readPackages } from "./libraries.mjs";

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

test("the plugin SDK family is not allowed to reach the notes library", () => {
  assert.deepEqual(LIBRARY_DEPENDENCIES["plugin-sdk-go"], ["protocol"]);
  const libraries = [
    library("libs/notes"),
    library("libs/protocol"),
    library("libs/plugin-sdk-go"),
    library("libs/plugin-sdk-go/host", ["libs/notes", "libs/notes/domain", "libs/protocol/gen/numen/v1"]),
  ];
  const found = findLibraryViolations(libraries, LIBRARY_DEPENDENCIES);
  assert.equal(found.length, 2);
  assert.match(found[0].symbol, /family plugin-sdk-go reaches notes/);
});

test("no go.mod of the plugin SDK requires the notes library", () => {
  const libraries = readLibraries().filter((one) => one.dir.startsWith("libs/plugin-sdk-go"));
  assert.ok(libraries.length >= 2);
  for (const one of libraries) {
    const notes = one.references.filter((reference) => reference.value.includes("/libs/notes"));
    assert.deepEqual(notes, [], `${one.dir} reaches the notes library`);
  }
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
  assert.deepEqual([...manager.declared].sort(), ["@numen/ui"]);
  assert.ok(manager.imports.length > 0);
  assert.deepEqual(findPackageViolations(packages, PACKAGE_DEPENDENCIES), []);
});

test("the rows allow the plugin manager the UI library, the plugin host the SDK and the core client, the SDK the schema and the UI build helpers, and the core client the schema", () => {
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-manager"], ["@numen/ui"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-host"], ["@numen/plugin-sdk", "@numen/core-client"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/plugin-sdk"], ["@numen/protocol", "@numen/ui"]);
  assert.deepEqual(PACKAGE_DEPENDENCIES["@numen/core-client"], ["@numen/protocol"]);
});

test("a plugin host that declares or imports the schema is reported", () => {
  const host = { dir: "libs/plugin-host-ts", name: "@numen/plugin-host", declared: ["@numen/plugin-sdk", "@numen/core-client", "@numen/protocol"], imports: [{ file: "libs/plugin-host-ts/src/a.ts", line: 2, name: "@numen/protocol" }] };
  const found = findPackageViolations([host], PACKAGE_DEPENDENCIES);
  assert.equal(found.length, 2);
  assert.match(found[0].symbol, /declares @numen\/protocol/);
  assert.match(found[1].symbol, /imports @numen\/protocol/);
});

test("a core client that declares or imports the SDK is reported", () => {
  const client = { dir: "libs/core-client", name: "@numen/core-client", declared: ["@numen/protocol", "@numen/plugin-sdk"], imports: [{ file: "libs/core-client/src/a.ts", line: 4, name: "@numen/plugin-sdk" }] };
  const found = findPackageViolations([client], PACKAGE_DEPENDENCIES);
  assert.equal(found.length, 2);
});

test("a plugin manager that declares or imports the protocol or the SDK is reported", () => {
  const manager = { dir: "libs/plugin-manager", name: "@numen/plugin-manager", declared: ["@numen/ui", "@numen/protocol"], imports: [{ file: "libs/plugin-manager/src/a.ts", line: 3, name: "@numen/plugin-sdk" }] };
  const found = findPackageViolations([manager], PACKAGE_DEPENDENCIES);
  assert.equal(found.length, 2);
  assert.match(found[0].symbol, /declares @numen\/protocol/);
  assert.match(found[1].symbol, /imports @numen\/plugin-sdk/);
});
