import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import { createBaseConfig } from "./base-eslint.mjs";

const numen = createBaseConfig(import.meta.dirname, { useProjectService: false }).find((entry) => entry.plugins?.numen).plugins.numen;

/** The message ids and lines the rule reports in the source, with at most 3 returned properties. */
function findReports(source, filename = "src/useA.ts") {
  const config = [{ files: ["**/*.ts"], plugins: { numen }, rules: { "numen/max-composable-return-properties": ["error", { max: 3 }] } }];
  return new Linter().verify(source, config, { filename }).map((message) => `${message.line} ${message.messageId ?? message.message}`);
}

describe("the composable return size rule", () => {
  it("fails an object literal returned directly with too many properties", () => {
    assert.deepEqual(findReports("function useA() { return { a: 1, b: 2, c: 3, d: 4 } }"), ["1 tooManyReturnedProperties"]);
    assert.deepEqual(findReports("const useA = () => ({ a: 1, b: 2, c: 3, d: 4 })"), ["1 tooManyReturnedProperties"]);
  });

  it("follows a returned identifier to the object literal it is declared with", () => {
    const source = ["function useA() {", "  const api = { a: 1, b: 2, c: 3, d: 4 }", "  return api", "}"].join("\n");
    assert.deepEqual(findReports(source), ["2 tooManyReturnedProperties"]);
    const arrow = ["const useB = () => {", "  const api = { a: 1, b: 2, c: 3, d: 4 }", "  return api", "}"].join("\n");
    assert.deepEqual(findReports(arrow), ["2 tooManyReturnedProperties"]);
  });

  it("fails a spread in the returned object, written in the return or in the returned variable", () => {
    assert.deepEqual(findReports("function useA(state) { return { ...state, a: 1 } }"), ["1 spreadInReturn"]);
    const source = ["function useA(state) {", "  const api = { ...state }", "  return api", "}"].join("\n");
    assert.deepEqual(findReports(source), ["2 spreadInReturn"]);
  });

  it("passes a small object and a function that is no composable", () => {
    assert.deepEqual(findReports("function useA() { const api = { a: 1, b: 2 }; return api }"), []);
    assert.deepEqual(findReports("function makeA(s) { return { ...s, a: 1, b: 2, c: 3, d: 4 } }"), []);
  });

  it("checks a create factory wherever it lives", () => {
    const source = "function createA() { return { a: 1, b: 2, c: 3, d: 4 } }";
    for (const file of ["src/entities/tab/model/a.ts", "src/app/a.ts", "src/lib/a.ts", "frontend/a.ts", "src/a.ts"]) {
      assert.deepEqual(findReports(source, file), ["1 tooManyReturnedProperties"]);
    }
    assert.deepEqual(findReports("const useA = (s) => ({ ...s })", "src/a.ts"), ["1 spreadInReturn"]);
    assert.deepEqual(findReports("function createA() { return { a: { b: 1, c: 2 }, d: 3 } }", "src/model/a.ts"), []);
  });

  it("fails a factory whose input object has more members than the limit", () => {
    assert.deepEqual(findReports("function createA({ a, b, c, d }) { return { a } }"), ["1 tooManyInputProperties"]);
  });

  it("passes a factory with a small input and a composable or helper with a large one", () => {
    assert.deepEqual(findReports("function createA({ a, b, c }) { return { a } }"), []);
    assert.deepEqual(findReports("function useA({ a, b, c, d }) { return { a } }"), []);
    assert.deepEqual(findReports("function buildA({ a, b, c, d }) { return { a } }"), []);
  });

  it("fails a factory that spreads a value it did not build, and passes one that spreads its own part", () => {
    assert.deepEqual(findReports("function createA(deps) { return { ...deps, a: 1 } }"), ["1 spreadOfInput"]);
    assert.deepEqual(findReports("function createA(deps) { return { ...deps.tabs, a: 1 } }"), ["1 spreadOfInput"]);
    assert.deepEqual(findReports("const createA = () => ({ ...makeParts(), a: 1 })"), ["1 spreadOfInput"]);
    const own = ["function createA() {", "  const part = { x: 1 }", "  return { ...part, a: 1 }", "}"].join("\n");
    assert.deepEqual(findReports(own), []);
  });

  it("fails a group of one key", () => {
    assert.deepEqual(findReports("function useA() { return { a: 1, group: { only: 2 } } }"), ["1 singleKeyGroup"]);
  });

  it("fails a group named for what it wraps, whatever it holds", () => {
    assert.deepEqual(findReports("function useA() { return { services: { open: 1, save: 2 }, a: 1 } }"), ["1 wrapperGroup"]);
    assert.deepEqual(findReports("function createA() { return { parts: { x: 1, y: 2 } } }"), ["1 wrapperGroup"]);
    assert.deepEqual(findReports("function useA() { return { drafts: { open: 1, save: 2 }, a: 1 } }"), []);
  });

  it("fails a group that only re-exposes members of the parts the function calls", () => {
    const source = [
      "function useA() {",
      "  const tabs = useTabs()",
      "  const { open, close } = useDialog()",
      "  return { group: { x: tabs.items, y: tabs.active, z: close, other: 1 } }",
      "}",
    ].join("\n");
    assert.deepEqual(findReports(source), ["4 reexposedGroup"]);
    const own = ["function useA() {", "  const tabs = useTabs()", "  return { group: { x: tabs.items, y: tabs.active, z: 1, other: 1 } }", "}"].join("\n");
    assert.deepEqual(findReports(own), []);
  });
});

const project = fs.mkdtempSync(path.join(os.tmpdir(), "lint-layers-composable-"));
fs.writeFileSync(
  path.join(project, "tsconfig.json"),
  JSON.stringify({ compilerOptions: { lib: ["ES2022", "DOM"], strict: true, target: "ES2022", module: "ESNext", skipLibCheck: true }, include: ["*.ts"] }),
);
after(() => fs.rmSync(project, { recursive: true, force: true }));

/** The message ids and lines the rule reports in a planted file, with at most 3 returned properties and type information. */
function findTypedReports(source) {
  const file = path.join(project, "useA.ts");
  fs.writeFileSync(file, source);
  const config = [
    {
      files: ["**/*.ts"],
      plugins: { numen },
      languageOptions: { parser: tseslint.parser, parserOptions: { projectService: true, tsconfigRootDir: project } },
      rules: { "numen/max-composable-return-properties": ["error", { max: 3 }] },
    },
  ];
  return new Linter({ cwd: project }).verify(source, config, { filename: file }).map((message) => `${message.line} ${message.messageId ?? message.message}`);
}

describe("the composable return size rule with type information", () => {
  it("counts the properties of a returned call result", () => {
    const source = ["const big = () => ({ a: () => 1, b: () => 2, c: () => 3, d: () => 4 })", "function useA() {", "  return big()", "}"].join("\n");
    assert.deepEqual(findTypedReports(source), ["3 tooManyReturnedProperties"]);
  });

  it("counts the properties of a value typed by an interface or a return type annotation", () => {
    const source = [
      "interface Api { a(): void; b(): void; c(): void; d(): void }",
      "declare const api: Api",
      "function useA(): Api { return api }",
      "const useB = () => api",
    ].join("\n");
    assert.deepEqual(findTypedReports(source), ["3 tooManyReturnedProperties", "4 tooManyReturnedProperties"]);
  });

  it("counts a factory that spreads by the type of what it returns, and fails a composable that spreads", () => {
    const source = ["const base = { a: () => 1, b: () => 2, c: () => 3 }", "const createA = () => ({ ...base, d: () => 4 })", "const useB = () => ({ ...base })"].join("\n");
    assert.deepEqual(findTypedReports(source), ["2 tooManyReturnedProperties", "2 spreadOfInput", "3 spreadInReturn"]);
  });

  it("counts the explicit properties of a factory that spreads once, inside the type", () => {
    const source = ["const createA = () => {", "  const part = { a: () => 1 }", "  return { ...part, b: () => 2, c: () => 3 }", "}"].join("\n");
    assert.deepEqual(findTypedReports(source), []);
  });

  it("counts the members of an inline, an interface and an alias input type", () => {
    assert.deepEqual(findTypedReports("const createA = (deps: { a: () => 1; b: () => 2; c: () => 3; d: () => 4 }) => ({ a: 1 })"), ["1 tooManyInputProperties"]);
    const alias = ["type Deps = { a: () => 1; b: () => 2; c: () => 3; d: () => 4 }", "const createA = (deps: Deps) => ({ a: 1 })"].join("\n");
    assert.deepEqual(findTypedReports(alias), ["2 tooManyInputProperties"]);
  });

  it("passes a words constant and a plain data record as an input or a returned value", () => {
    const words = "interface Words { a: string; b: string; c: string; d: string; e: string }";
    assert.deepEqual(findTypedReports([words, "const createA = (words: Words) => ({ a: 1 })"].join("\n")), []);
    const record = "interface Entry { a: string; b: { c: number; d: string[] }; e: boolean; f: null; g: 1 | 2 }";
    assert.deepEqual(findTypedReports([record, "const createA = (record: Entry) => ({ a: 1 })"].join("\n")), []);
    assert.deepEqual(findTypedReports([words, "declare const words: Words", "const createB = () => words"].join("\n")), []);
    assert.deepEqual(findTypedReports([words, "const createC = (words: Words) => ({ a: words, b: words, c: words, d: words })"].join("\n")), []);
  });

  it("does not count the data members of a bag that holds a few dependencies", () => {
    const source = [
      "interface Deps { read(): void; write(): void; a: string; b: string; c: string; d: string; e: number }",
      "const createA = (deps: Deps) => ({ a: 1 })",
    ].join("\n");
    assert.deepEqual(findTypedReports(source), []);
  });

  it("fails a bag of nine dependencies, whatever else it holds", () => {
    const members = Array.from({ length: 9 }, (_, index) => `d${index}(): void`).join("; ");
    assert.deepEqual(findTypedReports(`interface Deps { ${members} }\nconst createA = (deps: Deps) => ({ a: 1 })`), ["2 tooManyInputProperties"]);
    const mixed = `interface Deps { ${members}; words: { a: string } }\nconst createB = ({ d0, d1 }: Deps) => ({ a: 1 })`;
    assert.deepEqual(findTypedReports(mixed), ["2 tooManyInputProperties"]);
  });

  it("counts a service or a library type as a dependency, not as data", () => {
    const members = ["a: Map<string, string>", "b: Promise<void>", "c: Date", "d: () => void"].join("; ");
    assert.deepEqual(findTypedReports(`interface Deps { ${members} }\nconst createA = (deps: Deps) => ({ a: 1 })`), ["2 tooManyInputProperties"]);
  });

  it("counts the members of a declared input type", () => {
    const source = ["interface Deps { a: () => 1; b: () => 2; c: () => 3; d: () => 4 }", "declare const deps: Deps", "const createA = (input: Deps) => ({ a: 1 })"].join("\n");
    assert.deepEqual(findTypedReports(source), ["3 tooManyInputProperties"]);
  });

  it("passes a returned type from a library, an array and a function", () => {
    const source = ["function useA() { return new Map<string, number>() }", "function useB() { return [1, 2, 3, 4, 5] }", "function useC() { return () => 1 }"].join("\n");
    assert.deepEqual(findTypedReports(source), []);
  });
});
