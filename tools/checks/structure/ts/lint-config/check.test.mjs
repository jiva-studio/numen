import assert from "node:assert/strict";
import { test } from "node:test";
import {
  SCRIPT_STRICT_RULES,
  STRICT_RULES,
  findStrictRuleGaps,
  listLibProbes,
  listLintedPackages,
  pickProbeFiles,
  pickScriptProbeFiles,
} from "./check.mjs";

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

test("packages are derived from eslint configs and package.json files", () => {
  const tracked = [
    "package.json",
    "apps/mobile/eslint.config.js",
    "apps/landing/eslint.config.js",
    "apps/landing/package.json",
    "libs/ui/components/eslint.config.js",
    "libs/core/protocol/package.json",
    "plugins/notes/eslint.config.js",
    "plugins/notes/package.json",
    "plugins/readers-pdf/package.json",
    "plugins/readers-pdf/frontend/index.ts",
    "plugins/.example/eslint.config.js",
    "plugins/.example/package.json",
    "plugins/.draft/package.json",
    "plugins/agents-claude/Makefile",
  ];
  assert.deepEqual(listLintedPackages(tracked), [
    { dir: "apps/landing", hasConfig: true },
    { dir: "apps/mobile", hasConfig: true },
    { dir: "libs/core/protocol", hasConfig: false },
    { dir: "libs/ui/components", hasConfig: true },
    { dir: "plugins/.example", hasConfig: true },
    { dir: "plugins/notes", hasConfig: true },
    { dir: "plugins/readers-pdf", hasConfig: false },
  ]);
});

test("a package.json inside a configured directory is linted by that directory's config", () => {
  const tracked = ["tools/eslint.config.mjs", "tools/lint-layers/package.json", "tools/lint-layers/check.mjs"];
  assert.deepEqual(listLintedPackages(tracked), [{ dir: "tools", hasConfig: true }]);
});

test("the probes are the first source file and the first test file of the package", () => {
  const filesByPackage = {
    "plugins/notes": ["vite.config.ts", "frontend/index.d.ts", "frontend/a.stories.ts", "frontend/b.ts", "frontend/b.test.ts", "frontend/c.test.ts"],
    "plugins/notes-other": ["frontend/z.ts"],
  };
  const tracked = Object.entries(filesByPackage).flatMap(([dir, names]) => names.map((name) => `${dir}/${name}`));
  assert.deepEqual(pickProbeFiles(tracked, "plugins/notes"), ["plugins/notes/frontend/b.ts", "plugins/notes/frontend/b.test.ts"]);
  assert.deepEqual(pickProbeFiles(tracked, "plugins/missing"), ["plugins/missing/probe.ts", "plugins/missing/probe.test.ts"]);
});

test("a script package is checked against the script rule set only", () => {
  const rules = { "no-void": [2, { allowAsStatement: false }], "no-empty": [2, { allowEmptyCatch: false }], "no-empty-function": [2] };
  assert.deepEqual(findStrictRuleGaps(rules, SCRIPT_STRICT_RULES), []);
  assert.deepEqual(findStrictRuleGaps({ ...rules, "no-empty-function": [0] }, SCRIPT_STRICT_RULES), ["no-empty-function is not an error"]);
});

test("a config written as eslint.config.mjs makes its directory a linted package", () => {
  assert.deepEqual(listLintedPackages(["tools/eslint.config.mjs", "tools/lint-layers/check.mjs"]), [
    { dir: "tools", hasConfig: true },
  ]);
});

test("script probes are the first source and the first test, never a config", () => {
  const probes = pickScriptProbeFiles(
    ["tools/eslint.config.mjs", "tools/a/a.test.mjs", "tools/a/a.mjs", "tools/b/b.mjs", "libs/x/y.mjs"],
    "tools",
  );
  assert.deepEqual(probes, ["tools/a/a.mjs", "tools/a/a.test.mjs"]);
});

test("a lib segment is probed once per package, at its first source file, and tests and stories are not sources", () => {
  const tracked = [
    "apps/desktop/editor/eslint.config.js",
    "apps/desktop/editor/src/entities/tab/lib/a.test.ts",
    "apps/desktop/editor/src/entities/tab/lib/b.ts",
    "apps/desktop/editor/src/entities/tab/lib/c.ts",
    "plugins/notes/eslint.config.js",
    "plugins/notes/frontend/editor/lib/x.stories.ts",
    "plugins/notes/frontend/editor/lib/y.ts",
    "plugins/engines-ocr/eslint.config.js",
    "plugins/engines-ocr/frontend/ui/z.ts",
  ];
  assert.deepEqual(Array.from(listLibProbes(tracked)), [
    ["apps/desktop/editor", "apps/desktop/editor/src/entities/tab/lib/b.ts"],
    ["plugins/notes", "plugins/notes/frontend/editor/lib/y.ts"],
  ]);
});
