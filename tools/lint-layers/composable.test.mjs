import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
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

  it("checks a create factory in a model or app folder", () => {
    const source = "function createA() { return { a: 1, b: 2, c: 3, d: 4 } }";
    assert.deepEqual(findReports(source, "src/entities/tab/model/a.ts"), ["1 tooManyReturnedProperties"]);
    assert.deepEqual(findReports(source, "src/app/a.ts"), ["1 tooManyReturnedProperties"]);
    assert.deepEqual(findReports("const createA = (s) => ({ ...s })", "src/entities/tab/model/a.ts"), ["1 spreadInReturn"]);
    assert.deepEqual(findReports("function createA() { return { a: { b: 1, c: 2 }, d: 3 } }", "src/model/a.ts"), []);
  });

  it("leaves a create factory outside the model and app folders alone", () => {
    assert.deepEqual(findReports("function createA() { return { a: 1, b: 2, c: 3, d: 4 } }", "src/lib/a.ts"), []);
  });
});
