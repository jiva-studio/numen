import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import { createNoUnknownContractConfig } from "./base-eslint.mjs";

/** The lines the rule reports in the source. */
function findReports(source) {
  const config = [
    { files: ["**/*.ts"], languageOptions: { parser: tseslint.parser } },
    createNoUnknownContractConfig(["**/*.ts"]),
  ];
  return new Linter().verify(source, config, { filename: "src/contract.ts" }).map((message) => message.line);
}

describe("the unknown-in-contract rule", () => {
  it("fails unknown in an exported interface, function signature and alias", () => {
    assert.deepEqual(findReports("export interface A {\n  icon: unknown\n}"), [2]);
    assert.deepEqual(findReports("export function f(value: unknown): void {}"), [1]);
    assert.deepEqual(findReports("export type Value = unknown"), [1]);
    assert.deepEqual(findReports("export const f = (list: readonly unknown[]) => list.length"), [1]);
  });

  it("allows the one named alias of a caught value, and uses of it", () => {
    assert.deepEqual(findReports("export type Caught = unknown\nexport const f = (error: Caught) => String(error)"), []);
  });

  it("allows unknown in the body of an exported function and outside any export", () => {
    assert.deepEqual(findReports("export function f(): number {\n  const x: unknown = 1\n  return Number(x)\n}"), []);
    assert.deepEqual(findReports("interface Hidden {\n  value: unknown\n}"), []);
  });
});
