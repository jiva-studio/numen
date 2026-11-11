import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { getGoModReferences, getGoReferences, requireMatches } from "./graph.mjs";
describe("the selector match", () => {
  it("fails when a selector matched nothing and passes on one match", () => {
    assert.throws(() => requireMatches("files", 0), /selector matched nothing/);
    assert.doesNotThrow(() => requireMatches("files", 1));
  });
});

describe("the Go references", () => {
  it("reads the imports of a Go file and skips commented ones", () => {
    const source = 'package a\n\nimport (\n\t"fmt" // print\n\t// "os"\n\t/* "net" */\n)\n';
    assert.deepEqual(getGoReferences(source).imports, [{ line: 4, value: "fmt" }]);
  });
  it("reads the module paths of a go.mod and skips commented ones", () => {
    const source = "module github.com/jiva-studio/numen/a\n\nrequire github.com/jiva-studio/numen/b v0.0.0 // github.com/jiva-studio/numen/c\n";
    assert.deepEqual(getGoModReferences(source).imports, [{ line: 3, value: "github.com/jiva-studio/numen/b" }]);
  });
});
