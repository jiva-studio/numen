import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findGeometryReads } from "./check.mjs";

describe("the reads of a box or a style in a story", () => {
  it("finds a call, a destructured name and a matcher, and not a word in a string or a comment", () => {
    const text = [
      "const box = el.getBoundingClientRect()",
      "const style = getComputedStyle(el)",
      "await expect(el).toHaveStyle({ color: 'red' })",
      "const word = 'getBoundingClientRect'",
      "// getComputedStyle",
    ].join("\n");
    assert.deepEqual(findGeometryReads("a.stories.ts", text), [
      { name: "getBoundingClientRect", line: 1 },
      { name: "getComputedStyle", line: 2 },
      { name: "toHaveStyle", line: 3 },
    ]);
  });
});

describe("raw measurement in a story", () => {
  it("finds a bracketed browser method and a renamed UI measurement import", () => {
    const source = [
      'element["getBoundingClientRect"]()',
      'import { measureRect as bounds, measureElementRect } from "@numen/ui"',
      "const { getComputedStyle: style } = window",
    ].join("\n");
    assert.deepEqual(findGeometryReads("new.stories.ts", source), [
      { name: "getBoundingClientRect", line: 1 },
      { name: "measureRect", line: 2 },
      { name: "measureElementRect", line: 2 },
      { name: "getComputedStyle", line: 3 },
    ]);
  });
});
