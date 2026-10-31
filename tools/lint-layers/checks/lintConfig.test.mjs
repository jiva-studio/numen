import assert from "node:assert/strict";
import { test } from "node:test";
import { STRICT_RULES, findStrictRuleGaps, listLintedPackages, pickProbeFiles } from "./lintConfig.mjs";

/** The rules of a package that enforces the whole set, in the shape `eslint --print-config` prints. */
const strictRules = () => ({
  "no-void": [2, { allowAsStatement: false }],
  "no-empty": [2, { allowEmptyCatch: false }],
  "@typescript-eslint/no-empty-function": [2, { allow: [] }],
  "@typescript-eslint/no-floating-promises": [2, { ignoreVoid: false }],
  "@typescript-eslint/no-misused-promises": [2],
  "@typescript-eslint/await-thenable": [2],
  "@typescript-eslint/switch-exhaustiveness-check": [2, { considerDefaultExhaustiveForUnions: true }],
});

test("the whole rule set resolves without a gap", () => {
  assert.deepEqual(findStrictRuleGaps(strictRules()), []);
});

test("every rule of the set is reported when absent", () => {
  for (const { rule } of STRICT_RULES) {
    const rules = strictRules();
    delete rules[rule];
    assert.deepEqual(findStrictRuleGaps(rules), [`${rule} is not an error`]);
  }
});

test("a rule that is off or a warning is reported", () => {
  const rules = { ...strictRules(), "no-void": [0], "@typescript-eslint/no-misused-promises": [1] };
  assert.deepEqual(findStrictRuleGaps(rules), [
    "no-void is not an error",
    "@typescript-eslint/no-misused-promises is not an error",
  ]);
});

test("a severity written as a bare number or a name is read", () => {
  const rules = { ...strictRules(), "@typescript-eslint/await-thenable": 2, "@typescript-eslint/no-misused-promises": "error" };
  assert.deepEqual(findStrictRuleGaps(rules), []);
});

test("an option that loosens a rule is reported", () => {
  const rules = {
    ...strictRules(),
    "no-void": [2, { allowAsStatement: true }],
    "no-empty": [2, { allowEmptyCatch: true }],
    "@typescript-eslint/no-floating-promises": [2, { ignoreVoid: true }],
  };
  assert.equal(findStrictRuleGaps(rules).length, 3);
});

test("the default options of no-floating-promises are reported because they ignore void", () => {
  const rules = { ...strictRules(), "@typescript-eslint/no-floating-promises": [2] };
  assert.equal(findStrictRuleGaps(rules).length, 1);
});

test("an empty or missing rules object fails closed", () => {
  assert.equal(findStrictRuleGaps({}).length, STRICT_RULES.length);
  assert.deepEqual(findStrictRuleGaps(undefined), ["no resolved rules"]);
});

test("packages are derived from eslint configs and plugin package.json files", () => {
  const tracked = [
    "apps/mobile/eslint.config.js",
    "libs/ui/core/eslint.config.js",
    "plugins/notes/eslint.config.js",
    "plugins/notes/package.json",
    "plugins/readers-pdf/package.json",
    "plugins/readers-pdf/frontend/index.ts",
    "plugins/.example/eslint.config.js",
    "plugins/agents-claude/Makefile",
  ];
  const packages = listLintedPackages(tracked, (dir) => ["plugins/notes", "plugins/readers-pdf"].includes(dir));
  assert.deepEqual(packages, [
    { dir: "apps/mobile", hasConfig: true },
    { dir: "libs/ui/core", hasConfig: true },
    { dir: "plugins/.example", hasConfig: true },
    { dir: "plugins/notes", hasConfig: true },
    { dir: "plugins/readers-pdf", hasConfig: false },
  ]);
});

test("the probes are the first source file and the first test file of the package", () => {
  const tracked = [
    "plugins/notes/vite.config.ts",
    "plugins/notes/frontend/index.d.ts",
    "plugins/notes/frontend/a.stories.ts",
    "plugins/notes/frontend/b.ts",
    "plugins/notes/frontend/b.test.ts",
    "plugins/notes/frontend/c.test.ts",
    "plugins/notes-other/frontend/z.ts",
  ];
  assert.deepEqual(pickProbeFiles(tracked, "plugins/notes"), ["plugins/notes/frontend/b.ts", "plugins/notes/frontend/b.test.ts"]);
  assert.deepEqual(pickProbeFiles(tracked, "plugins/missing"), ["plugins/missing/probe.ts", "plugins/missing/probe.test.ts"]);
});
