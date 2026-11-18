import assert from "node:assert/strict";
import { test } from "node:test";
import { getOwners, parseCodeowners } from "./codeowners.mjs";

const rules = parseCodeowners(`
# a comment
* @Owner
/libs/ui/ @owner @ui-lead
*.md @docs-lead
/.github/CODEOWNERS @owner
apps/**/generated.go
`);

test("a file takes the owners of the last rule that matches", () => {
  assert.deepEqual(getOwners(rules, "main.go"), ["owner"]);
  assert.deepEqual(getOwners(rules, "libs/ui/components/a.ts"), ["owner", "ui-lead"]);
  assert.deepEqual(getOwners(rules, "docs/readme.md"), ["docs-lead"]);
  assert.deepEqual(getOwners(rules, ".github/CODEOWNERS"), ["owner"]);
});

test("a rule without owners clears the owners of the earlier rules", () => {
  assert.deepEqual(getOwners(rules, "apps/desktop/generated.go"), []);
});

test("a rule that starts with a slash does not match deeper paths", () => {
  assert.deepEqual(getOwners(rules, "other/libs/ui/a.ts"), ["owner"]);
});

test("a file without a matching rule has no owner", () => {
  assert.deepEqual(getOwners(parseCodeowners("/docs/ @owner"), "src/a.go"), []);
});
