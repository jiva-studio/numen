import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  findNonCamelCaseJsonKeys,
  findPluginCrossings,
  findPluginSchemas,
  findStrykerFindings,
  findStylelintSuppressions,
  findSuppressions,
  findUnassignedPaths,
  findWorkspaceGaps,
  isGenerated,
  parseWorkspaceUses,
} from "./check.mjs";
describe("plugin isolation over the captured plugin", () => {
  const packages = new Map([["alpha", "@example/alpha"], ["beta", "@example/beta"]]);
  const base = [
    { file: "plugins/alpha/backend/a.go", text: 'import "github.com/jiva-studio/numen/plugins/alpha/backend/b"' },
    { file: "plugins/beta/backend/b.go", text: "package b" },
  ];
  it("accepts references to the own plugin and ignores comments", () => {
    const withComment = [
      ...base,
      { file: "plugins/alpha/backend/a.go", text: '// github.com/jiva-studio/numen/plugins/beta/backend\npackage a\n' },
    ];
    assert.deepEqual(findPluginCrossings(withComment, packages), []);
  });
  it("finds a Go import of another plugin", () => {
    const sources = [...base, { file: "plugins/alpha/backend/a.go", text: 'import "github.com/jiva-studio/numen/plugins/beta/backend"' }];
    assert.equal(findPluginCrossings(sources, packages).length, 1);
  });
  it("finds a dependency on the package of another plugin", () => {
    const sources = [...base, { file: "plugins/alpha/package.json", text: JSON.stringify({ dependencies: { "@example/beta": "1.0.0" } }) }];
    assert.equal(findPluginCrossings(sources, packages).length, 1);
  });
});

describe("the schemas of plugins", () => {
  it("finds a protobuf schema under a plugin and none elsewhere", () => {
    const files = ["plugins/alpha/proto/alpha.proto", "plugins/alpha/backend/a.go", "libs/shared/protocol/proto/numen/v1/a.proto"];
    assert.deepEqual(findPluginSchemas(files).map((one) => one.file), ["plugins/alpha/proto/alpha.proto"]);
  });
});

describe("the stylelint suppressions", () => {
  const found = (file, text) => findStylelintSuppressions([{ file, text }]).map((one) => one.symbol);
  it("finds an ignore setting in the configuration and a disable comment in a stylesheet or component", () => {
    assert.deepEqual(found("stylelint.config.mjs", 'export default {\n  ignoreFiles: ["a/**"],\n}'), ["stylelint ignore setting"]);
    assert.deepEqual(found("a.css", "/* stylelint-disable color-no-hex */\na { color: #fff }"), ["stylelint-disable comment"]);
    assert.deepEqual(found("A.vue", "<style>\n/* stylelint-disable-next-line x */\n</style>"), ["stylelint-disable comment"]);
    assert.deepEqual(found(".stylelintignore", "dist\n"), [".stylelintignore"]);
  });
  it("passes a configuration with no ignore setting", () => {
    assert.deepEqual(found("stylelint.config.mjs", "export default { rules: {} }"), []);
  });
});

describe("the Stryker disable comments", () => {
  const find = (file, text, cap) => findStrykerFindings([{ file, text }], cap).map((one) => one.symbol);
  it("fails a disable without a reason and passes one with a sentence", () => {
    assert.deepEqual(find("a.ts", "// Stryker disable next-line StringLiteral\nconst a = 'x'"), ["Stryker disable without a reason"]);
    assert.deepEqual(find("a.ts", "// Stryker disable next-line StringLiteral: label of the interface\nconst a = 'x'"), []);
    assert.deepEqual(find("a.ts", "/* Stryker disable OptionalChaining,ObjectLiteral: forwards optional capabilities */\nconst a = 1"), []);
  });
  it("fails more disables than the cap in shipped code and counts no test", () => {
    const text = "// Stryker disable next-line StringLiteral: label of the interface\nconst a = 'x'";
    assert.equal(find("a.ts", text, 0).length, 1);
    assert.deepEqual(find("a.test.ts", text, 0), []);
  });
  it("fails shipped code above the recorded Stryker ceiling", () => {
    const text = ("// Stryker disable StringLiteral: label of the interface\nconst value = 'x';\n").repeat(215);
    assert.equal(find("a.ts", text).length, 1);
  });
});

describe("the suppressions", () => {
  const found = (file, text) => findSuppressions([{ file, text }]).map((one) => one.symbol);
  it("finds a Go suppression comment", () => {
    assert.deepEqual(found("a.go", "package a\nfunc f() {} //nolint:errcheck\n"), ["nolint:errcheck"]);
    assert.deepEqual(found("a.go", "package a\n/* nolint */\n"), ["nolint"]);
  });
  it("finds TypeScript suppression comments", () => {
    assert.deepEqual(found("a.ts", "// eslint-disable-next-line no-console\nconsole.log(1)"), ["eslint-disable-next-line"]);
    assert.deepEqual(found("a.ts", "/* eslint-disable */\nconst a = 1"), ["eslint-disable"]);
    assert.deepEqual(found("a.ts", "// @ts-expect-error\nconst a: number = ''"), ["@ts-expect-error"]);
    assert.deepEqual(found("a.ts", "const a = 1 // @ts-ignore"), ["@ts-ignore"]);
  });
  it("finds a suppression in a component script and template", () => {
    assert.equal(found("A.vue", "<template><!-- eslint-disable vue/no-v-html --><p /></template>").length, 1);
    assert.equal(found("A.vue", "<script setup lang=\"ts\">\n// @ts-ignore\nconst a = 1\n</script>").length, 1);
  });
  it("finds the exclusion keys of a golangci configuration", () => {
    const text = "linters:\n  exclusions:\n    rules: []\n  settings:\n    errcheck:\n      exclude-functions:\n        - os.Remove\n";
    assert.deepEqual(found(".golangci.yml", text), ["exclusions", "exclude-functions"]);
  });
  it("finds nolint in a comment of a golangci configuration", () => {
    assert.deepEqual(found(".golangci.yml", "linters:\n  # skip with nolint here\n"), ["nolint"]);
    assert.deepEqual(found(".golangci.yml", "run: # a # b nolint\n"), ["nolint"]);
    assert.deepEqual(found(".golangci.yml", "nolint: true # comment\n  # nolintx\n"), []);
  });
  it("finds a file named in an eslint ignores or files entry", () => {
    assert.equal(found("eslint.config.js", "export default [{ ignores: ['**/clock.ts', 'dist/**'] }]").length, 1);
    assert.equal(found("eslint.config.js", "export default [{ files: ['src/app/thing.ts'] }]").length, 1);
    assert.equal(found("eslint.config.js", "export default [{ files: ['**/*.ts'], ignores: ['dist/**'] }]").length, 0);
  });
  it("finds a list of repository files in a check, in a constant or inline", () => {
    assert.equal(found("tools/checks/structure/ts/x/check.mjs", 'const EXEMPT = ["apps/desktop/a.ts", "libs/x/b.go"]').length, 1);
    assert.equal(found("tools/checks/structure/ts/x/check.mjs", 'for (const one of ["plugins/vite.shared.ts", "plugins/vitest.config.ts"]) check(one)').length, 1);
    assert.equal(found("tools/checks/structure/ts/x/check.test.mjs", 'const fixture = ["apps/desktop/a.ts"]').length, 0);
    assert.equal(found("tools/checks/structure/ts/x/check.mjs", 'const TOP = ["apps", "libs", "plugins"]').length, 0);
  });
  it("finds a constant of a check that lists packages, by folder or by Go import path", () => {
    assert.deepEqual(found("tools/checks/structure/ts/x/check.mjs", 'const ROOTS = ["apps/desktop", "libs/x"]'), ["package list ROOTS"]);
    assert.deepEqual(found("tools/checks/style/x/check.mjs", "const SKIPPED = new Set(['apps/landing'])"), ["package list SKIPPED"]);
    assert.deepEqual(found("tools/lint-layers/x.mjs", 'const DRIVING = ["github.com/jiva-studio/numen/libs/core/kernel/adapter/cli"]'), ["package list DRIVING"]);
    assert.deepEqual(found("tools/checks/structure/ts/x/check.mjs", 'const GLOBS = ["apps/**/*.ts", "libs/x"]'), []);
    assert.deepEqual(found("tools/checks/structure/ts/x/check.mjs", 'run(["apps/desktop", "libs/x"])'), []);
  });
  it("finds a list of repository files in the JSON data of a check", () => {
    const data = (text) => found("tools/checks/structure/ts/x/allowed.json", text);
    assert.deepEqual(data('["apps/desktop/a.ts", "libs/x/b.go"]'), ["file list (root)"]);
    assert.deepEqual(data('[{ "file": "plugins/p/frontend/A.stories.ts", "reason": "it is drawn" }]'), ["file list (root)"]);
    assert.deepEqual(data('{ "exempt": { "files": ["libs/x/b.go"] } }'), ["file list exempt.files"]);
    assert.deepEqual(data('{ "go": { "libs/shared/note": ["libs/shared/atomicfile"] }, "words": ["get", "set"] }'), []);
    assert.deepEqual(found("tools/checks/rules.json", '[{ "paths": ["tools/lint-layers/layers.mjs"] }]'), []);
  });
  it("keeps a list of JSON data whose keptBy names an issue and a reason, and fails one without", () => {
    const data = (text) => found("tools/checks/structure/ts/x/allowed.json", text);
    const entries = '[{ "file": "plugins/p/frontend/A.stories.ts", "reason": "it is drawn" }]';
    assert.deepEqual(data(`{ "keptBy": "#1949 the stories stay listed", "stories": ${entries} }`), []);
    assert.deepEqual(data(`{ "stories": ${entries} }`), ["file list stories"]);
    assert.deepEqual(data(`{ "keptBy": "the stories stay listed", "stories": ${entries} }`), ["keptBy names no issue and reason"]);
    assert.deepEqual(data(`{ "keptBy": "#1949", "stories": ${entries} }`), ["keptBy names no issue and reason"]);
  });
  it("keeps a list of a check with a kept-list marker and an issue right above it, and fails an unmarked one", () => {
    const check = (text) => found("tools/checks/structure/ts/x/check.mjs", text);
    const list = 'const ROOTS = ["apps/desktop", "libs/x"];\n';
    assert.deepEqual(check(`// numen:kept-list #2126 the catalog stays listed\n${list}`), []);
    assert.deepEqual(check(`/**\n * numen:kept-list #2126 the catalog stays listed\n */\n${list}`), []);
    assert.deepEqual(check(list), ["package list ROOTS"]);
    assert.deepEqual(check(`// numen:kept-list #2126 the catalog stays listed\n\n${list}`), ["package list ROOTS"]);
    assert.deepEqual(check(`// the catalog stays listed\n${list}`), ["package list ROOTS"]);
    assert.deepEqual(check(`// numen:kept-list the catalog stays listed\n${list}`), ["a kept-list marker names no issue and reason", "package list ROOTS"]);
  });
  it("fails a kept-list marker without an issue in a Go file", () => {
    assert.deepEqual(found("tools/checks/structure/go/x/rule_test.go", "package x\n\n// numen:kept-list #1957 the bridge is kept\nvar a = 1\n"), []);
    assert.deepEqual(found("tools/checks/structure/go/x/rule_test.go", "package x\n\n// numen:kept-list\nvar a = 1\n"), ["a kept-list marker names no issue and reason"]);
  });
  it("does not read a marker inside a string, or generated code", () => {
    assert.deepEqual(found("a.ts", "const a = '// eslint-disable-next-line'"), []);
    assert.deepEqual(found("a.go", "// Code generated by protoc. DO NOT EDIT.\n\npackage a\n//nolint\n"), []);
    assert.ok(isGenerated("// Code generated by x. DO NOT EDIT.\npackage a"));
    assert.equal(isGenerated("package a\n// Code generated by x. DO NOT EDIT."), true);
    assert.equal(isGenerated("\n\t# @generated\n"), true);
    assert.equal(isGenerated("x // Code generated by x. DO NOT EDIT."), false);
  });
});

describe("the paths that no layer owns", () => {
  it("finds a module outside the layers", () => {
    assert.equal(findUnassignedPaths(["stray/go.mod", "libs/core/kernel/go.mod", "tools/x/package.json"]).length, 1);
  });
  it("finds a file directly in a layer root", () => {
    assert.equal(findUnassignedPaths(["plugins/notes.txt", "plugins/.gitignore"]).length, 1);
  });
  it("finds a plugin directory without a manifest", () => {
    const files = ["plugins/a/manifest.json", "plugins/a/x.ts", "plugins/b/x.ts", "plugins/.example/x.ts"];
    assert.deepEqual(findUnassignedPaths(files).map((one) => one.symbol), ["plugin b has no manifest.json"]);
  });
});

describe("the go workspace", () => {
  const work = "go 1.26\n\nuse (\n\t./libs/core/kernel\n\t./plugins/a\n)\n\nreplace x => ./libs/core/kernel\n";
  it("lists the used directories", () => {
    assert.deepEqual(parseWorkspaceUses(work), ["libs/core/kernel", "plugins/a"]);
    assert.deepEqual(parseWorkspaceUses("use ./single\n"), ["single"]);
    assert.deepEqual(parseWorkspaceUses("use ( // modules\n\t./a // first\n)\nuse ./b//c\n"), ["a", "b"]);
  });
  it("finds a module the workspace leaves out and a use without a module", () => {
    const gaps = findWorkspaceGaps(work, ["libs/core/kernel", "plugins/b"]).map((one) => one.symbol);
    assert.deepEqual(gaps, ["missing use ./plugins/b", "use ./plugins/a has no go.mod"]);
  });
  it("accepts a workspace that matches", () => {
    assert.deepEqual(findWorkspaceGaps(work, ["libs/core/kernel", "plugins/a"]), []);
  });
});

describe("the camelCase JSON keys", () => {
  it("rejects a snake_case key in a manifest JSON schema and accepts camelCase", () => {
    const badManifest = JSON.stringify({
      id: "md.numen.example",
      settings: {
        schema: {
          type: "object",
          properties: {
            maxTokens: { type: "number" },
            chunk_tokens: { type: "number" },
          },
        },
      },
    });
    const findings = findNonCamelCaseJsonKeys([{ file: "plugins/example/manifest.json", text: badManifest }]);
    assert.deepEqual(findings.map((one) => one.symbol), ['JSON key "chunk_tokens"']);
  });
  it("rejects a snake_case or PascalCase property in a TypeScript JSON shape and accepts RPC method entries", () => {
    const source = [
      "export interface StatsOut { totalCount: number; max_side: number; Answered: number }",
      "export type Methods = { GetStats: { request: Record<string, never>; response: StatsOut } }",
    ].join("\n");
    const findings = findNonCamelCaseJsonKeys([{ file: "plugins/example/frontend/api/stats.ts", text: source }]);
    assert.deepEqual(findings.map((one) => one.symbol), ['JSON property "max_side"', 'JSON property "Answered"']);
  });
});
