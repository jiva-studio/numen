import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import { createClockSelectors, createPureSegmentConfig } from "./base-eslint.mjs";

/** Lines of the clock violations that the selectors report in the source. */
function findClockLines(source, clockFunction, monotonicFunction) {
  const config = [{ rules: { "no-restricted-syntax": ["error", ...createClockSelectors(clockFunction, monotonicFunction)] } }];
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

describe("the monotonic clock selectors", () => {
  it("fail performance.now() outside the sanctioned function", () => {
    const source = ["function getMonotonicNow() { return performance.now() }", "const stamp = () => performance.now()"].join("\n");
    assert.deepEqual(findClockLines(source, "getNow", "getMonotonicNow"), [2]);
  });

  it("fail performance.now() when no monotonic function is named", () => {
    assert.deepEqual(findClockLines("const a = performance.now()"), [1]);
  });
});

describe("the pure segment config", () => {
  const config = createPureSegmentConfig({ clockFunction: "getNow", monotonicFunction: "getMonotonicNow" });

  /** Lines of the violations in the source, linted as a file the config applies to. */
  function findPureLines(source, filename) {
    return new Linter().verify(source, [config], { filename }).filter((message) => message.ruleId).map((message) => message.line);
  }

  it("fails setTimeout, setInterval and a timer reached through a window", () => {
    const source = ["setTimeout(run, 1)", "setInterval(run, 1)", "win.setTimeout(run, 1)"].join("\n");
    assert.deepEqual(findPureLines(source, "src/shared/lib/wait.ts"), [1, 2, 3]);
  });

  it("fails the clocks and randomness", () => {
    assert.deepEqual(findPureLines("Date.now()\nperformance.now()\nMath.random()", "src/domain/pick.ts"), [1, 2, 3]);
  });

  it("applies to nothing outside lib/ and domain/ and to no test", () => {
    assert.deepEqual(findPureLines("setTimeout(run, 1)", "src/features/tree/model/debounce.ts"), []);
    assert.deepEqual(findPureLines("setTimeout(run, 1)", "src/shared/lib/wait.test.ts"), []);
  });
});
