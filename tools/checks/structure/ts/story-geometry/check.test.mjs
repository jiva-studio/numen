import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findGeometryReads, findGeometryReadsOutsideAllowlist } from "./check.mjs";

const symbols = (found) => found.map((one) => `${one.file}:${one.line}`);

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

describe("the allow-list of the stories that prove geometry", () => {
  const reading = "el.getBoundingClientRect()\n";

  it("fails a story file that reads a box and is not allowed", () => {
    assert.deepEqual(symbols(findGeometryReadsOutsideAllowlist([["a.stories.ts", reading]], [])), ["a.stories.ts:1"]);
  });

  it("passes an allowed file that reads a box and says why", () => {
    assert.deepEqual(findGeometryReadsOutsideAllowlist([["a.stories.ts", reading]], [{ file: "a.stories.ts", reason: "the menu stays inside the viewport" }]), []);
  });

  it("fails an allowed file with no reason, one that reads nothing and one that is not a story file", () => {
    const found = findGeometryReadsOutsideAllowlist(
      [["a.stories.ts", reading], ["b.stories.ts", "const x = 1\n"]],
      [{ file: "a.stories.ts", reason: " " }, { file: "b.stories.ts", reason: "inside" }, { file: "c.stories.ts", reason: "inside" }],
    );
    assert.deepEqual(symbols(found), ["a.stories.ts:1", "b.stories.ts:1", "c.stories.ts:1"]);
  });
});
