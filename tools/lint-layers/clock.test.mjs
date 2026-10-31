import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import { createClockSelectors } from "./base-eslint.mjs";

/** Lines of the clock violations that the selectors report in the source. */
function findClockLines(source, clockFunction) {
  const config = [{ rules: { "no-restricted-syntax": ["error", ...createClockSelectors(clockFunction)] } }];
  return new Linter().verify(source, config).map((message) => message.line);
}

describe("the wall clock selectors", () => {
  it("fail Date.now() and new Date() in any function when no clock function is named", () => {
    assert.deepEqual(findClockLines("function getNow() { return Date.now() }"), [1]);
    assert.deepEqual(findClockLines("const a = () => new Date()"), [1]);
  });

  it("pass the declaration of the sanctioned clock function", () => {
    assert.deepEqual(findClockLines("function getNow() { return Date.now() }", "getNow"), []);
    assert.deepEqual(findClockLines("function getNow() { return new Date().getTime() }", "getNow"), []);
  });

  it("fail a stray Date.now() or new Date() outside the sanctioned function", () => {
    const source = [
      "function getNow() { return Date.now() }",
      "function stamp() { return Date.now() }",
      "const today = () => new Date()",
      "class Box { read() { return Date.now() } }",
    ].join("\n");
    assert.deepEqual(findClockLines(source, "getNow"), [2, 3, 4]);
  });

  it("pass a date built from a value", () => {
    assert.deepEqual(findClockLines("const a = (n) => new Date(n)", "getNow"), []);
  });

  it("fail a function named like the sanctioned one when it is an arrow or a method", () => {
    assert.deepEqual(findClockLines("const getNow = () => Date.now()", "getNow"), [1]);
    assert.deepEqual(findClockLines("const o = { getNow() { return Date.now() } }", "getNow"), [1]);
  });
});
