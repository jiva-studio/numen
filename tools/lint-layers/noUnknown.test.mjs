import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import { numenPlugin } from "./base-eslint.mjs";

/** The lines the rule reports in the source. */
function findReports(source) {
  const config = [
    {
      files: ["**/*.ts"],
      languageOptions: { parser: tseslint.parser },
      plugins: { numen: numenPlugin },
      rules: { "numen/no-unknown": "error" },
    },
  ];
  return new Linter().verify(source, config, { filename: "src/contract.ts" }).map((message) => message.line);
}

describe("the no-unknown rule", () => {
  it("fails unknown in an exported interface, function signature and alias", () => {
    assert.deepEqual(findReports("export interface A {\n  icon: unknown\n}"), [2]);
    assert.deepEqual(findReports("export function f(value: unknown): void {}"), [1]);
    assert.deepEqual(findReports("export type Value = unknown"), [1]);
    assert.deepEqual(findReports("export const f = (list: readonly unknown[]) => list.length"), [1]);
  });

  it("fails unknown in a declaration that is not exported and in a function body", () => {
    assert.deepEqual(findReports("interface Hidden {\n  value: unknown\n}"), [2]);
    assert.deepEqual(findReports("export function f(): number {\n  const x: unknown = 1\n  return Number(x)\n}"), [2]);
    assert.deepEqual(findReports("function g(run: () => Promise<unknown>): void {}"), [1]);
  });

  it("fails an index signature and a record of unknown", () => {
    assert.deepEqual(findReports("interface A {\n  [key: string]: unknown\n}"), [2]);
    assert.deepEqual(findReports("const a: Record<string, unknown> = {}"), [1]);
  });

  it("fails a double assertion through unknown", () => {
    assert.deepEqual(findReports("const a = 1 as unknown as string"), [1]);
  });

  it("allows the two named aliases, and uses of Caught", () => {
    assert.deepEqual(findReports("export type Caught = unknown\nexport const f = (error: Caught) => String(error)"), []);
    assert.deepEqual(findReports("type Unvalidated = unknown"), []);
  });

  it("allows Unvalidated as the parameter of a parse function or a type guard that narrows it", () => {
    assert.deepEqual(findReports("const parseCount = (value: Unvalidated): number => (typeof value === 'number' ? value : 0)"), []);
    assert.deepEqual(findReports("function parseCount(value: Unvalidated): number {\n  return isCount(value) ? value : 0\n}"), []);
    assert.deepEqual(findReports("const isCount = (value: Unvalidated): value is number => typeof value === 'number'"), []);
    assert.deepEqual(findReports("const parseUser = (value: Unvalidated) => schema.parse(value)"), []);
    assert.deepEqual(findReports("const parseList = (value: Unvalidated) => (Array.isArray(value) ? value : [])"), []);
    assert.deepEqual(findReports("const parseBox = (value: Unvalidated) => ('size' in value ? 1 : 0)"), []);
    assert.deepEqual(findReports("const parseWhen = (value: Unvalidated) => (value instanceof Date ? value : undefined)"), []);
  });

  it("fails an Unvalidated parameter that the body never narrows", () => {
    assert.deepEqual(findReports("const parseCount = (value: Unvalidated): number => Number(value)"), [1]);
    assert.deepEqual(findReports("function parseCount(value: Unvalidated): number {\n  return value as number\n}"), [1]);
    assert.deepEqual(findReports("const isCount = (value: Unvalidated): value is number => true"), [1]);
    assert.deepEqual(findReports("const parseCount = (value: Unvalidated, other: number) => (typeof other === 'number' ? 1 : 0)"), [1]);
  });

  it("allows Caught as the parameter of a catch clause or of an error callback", () => {
    assert.deepEqual(findReports("try { run() } catch (error: Caught) { report(error) }"), []);
    assert.deepEqual(findReports("load().catch((err: Caught) => report(err))"), []);
    assert.deepEqual(findReports("interface Deps {\n  reportError(error: Caught): void\n  onError: (reason: Caught) => void\n}"), []);
    assert.deepEqual(findReports("type Fail = (kind: string, error: Caught) => void"), []);
    assert.deepEqual(findReports("function describe(cause: Caught): string { return String(cause) }"), []);
  });

  it("fails Caught anywhere else", () => {
    assert.deepEqual(findReports("interface State {\n  last: Caught\n}"), [2]);
    assert.deepEqual(findReports("const failures: Caught[] = []"), [1]);
    assert.deepEqual(findReports("const read = (value: Caught) => String(value)"), [1]);
    assert.deepEqual(findReports("const run = (): Promise<Caught> => Promise.resolve(1)"), [1]);
    assert.deepEqual(findReports("let held: Caught"), [1]);
  });

  it("fails Unvalidated anywhere else", () => {
    assert.deepEqual(findReports("const read = (value: Unvalidated): number => Number(value)"), [1]);
    assert.deepEqual(findReports("interface A {\n  value: Unvalidated\n}"), [2]);
    assert.deepEqual(findReports("type Parse = (value: Unvalidated) => number"), [1]);
  });
});
