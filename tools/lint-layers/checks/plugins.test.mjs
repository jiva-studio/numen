import assert from "node:assert/strict";
import { test } from "node:test";
import { findPluginGoModViolations } from "./plugins.mjs";

const ROOT = "github.com/jiva-studio/numen";

const CLEAN = `module ${ROOT}/plugins/example

go 1.26.9

require (
	${ROOT}/libs/plugin-sdk-go v0.0.0
	${ROOT}/libs/plugin-sdk-go/host v0.0.0
)
`;

test("a plugin go.mod that requires only the SDK is clean", () => {
  assert.deepEqual(findPluginGoModViolations("plugins/example/go.mod", CLEAN), []);
});

test("a plugin go.mod that requires the notes library is reported with the line", () => {
  const text = CLEAN.replace("\n)\n", `\n\t${ROOT}/libs/notes v0.0.0\n)\n`);
  const found = findPluginGoModViolations("plugins/example/go.mod", text);
  assert.equal(found.length, 1);
  assert.match(found[0].symbol, /libs\/notes in go\.mod/);
  assert.equal(found[0].line, 8);
});

test("a plugin go.mod that replaces the notes library by a relative path is reported twice", () => {
  const text = `${CLEAN}\nreplace ${ROOT}/libs/notes => ../../libs/notes\n`;
  const found = findPluginGoModViolations("plugins/example/go.mod", text);
  assert.equal(found.length, 2);
});

test("a comment that names the notes library is not a requirement", () => {
  const text = `${CLEAN}\n// ${ROOT}/libs/notes is not required\n`;
  assert.deepEqual(findPluginGoModViolations("plugins/example/go.mod", text), []);
});
