import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  findNonCamelCaseJsonKeys,
  findPluginCrossings,
  findPluginSchemas,
  findSuppressions,
  findUnassignedPaths,
  findWorkspaceGaps,
  isGenerated,
  parseWorkspaceUses,
} from "./checks/discipline.mjs";
import { findTsPackagesWithoutEslint } from "./checks/frontend.mjs";
import { getGoModReferences, getGoReferences, removeLineComment, requireMatches } from "./checks/helpers.mjs";
import { compareGoVersions, getLinterBuildGap, parseBuiltWithGo, parseGoDirective } from "./checks/tools.mjs";

describe("the minimum match count", () => {
  it("fails on zero and on too few", () => {
    assert.throws(() => requireMatches("files", 0, 1), /at least 1/);
    assert.throws(() => requireMatches("files", 2, 3), /at least 3/);
    assert.doesNotThrow(() => requireMatches("files", 3, 3));
  });
});

describe("plugin isolation over the captured plugin", () => {
  const packages = new Map([["alpha", "@example/alpha"], ["beta", "@example/beta"]]);
  const base = [
    { file: "plugins/alpha/backend/a.go", text: 'import "github.com/jiva-studio/numen/plugins/alpha/backend/b"' },
    { file: "plugins/beta/backend/b.go", text: "package b" },
  ];
  it("accepts references to the own plugin", () => {
    assert.deepEqual(findPluginCrossings(base, packages), []);
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

describe("the TypeScript packages without a lint configuration", () => {
  it("finds a package that has no eslint.config.js", () => {
    const found = findTsPackagesWithoutEslint(["libs/a", "apps/b"], (dir) => dir === "libs/a");
    assert.deepEqual(found.map((one) => one.file), ["apps/b/tsconfig.json"]);
  });
});

describe("the schemas of plugins", () => {
  it("finds a protobuf schema under a plugin and none elsewhere", () => {
    const files = ["plugins/alpha/proto/alpha.proto", "plugins/alpha/backend/a.go", "libs/protocol/proto/numen/v1/a.proto"];
    assert.deepEqual(findPluginSchemas(files).map((one) => one.file), ["plugins/alpha/proto/alpha.proto"]);
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
    assert.equal(found("tools/lint-layers/checks/x.mjs", 'const EXEMPT = ["apps/desktop/a.ts", "libs/x/b.go"]').length, 1);
    assert.equal(found("tools/lint-layers/checks/x.mjs", 'for (const one of ["plugins/vite.shared.ts", "plugins/vitest.config.ts"]) check(one)').length, 1);
    assert.equal(found("tools/lint-layers/checks/x.test.mjs", 'const fixture = ["apps/desktop/a.ts"]').length, 0);
    assert.equal(found("tools/lint-layers/checks/x.mjs", 'const ROOTS = ["apps/desktop", "libs/x"]').length, 0);
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
    assert.equal(findUnassignedPaths(["stray/go.mod", "libs/core/go.mod", "tools/x/package.json"]).length, 1);
  });
  it("finds a file directly in a layer root", () => {
    assert.equal(findUnassignedPaths(["plugins/notes.txt", "plugins/.gitignore"]).length, 1);
  });
  it("finds a plugin directory without a manifest", () => {
    const files = ["plugins/a/manifest.json", "plugins/a/x.ts", "plugins/b/x.ts", "plugins/.example/x.ts"];
    assert.deepEqual(findUnassignedPaths(files).map((one) => one.symbol), ["plugin b has no manifest.json"]);
  });
});

describe("the Go references", () => {
  it("drops the text from the first // on", () => {
    assert.equal(removeLineComment('a := "x" // b // c'), 'a := "x" ');
    assert.equal(removeLineComment("plain"), "plain");
  });
  it("reads the imports of a Go file and skips commented ones", () => {
    const source = 'package a\n\nimport (\n\t"fmt" // print\n\t// "os"\n)\n';
    assert.deepEqual(getGoReferences(source).imports, [{ line: 4, value: "fmt" }]);
  });
  it("reads the module paths of a go.mod and skips commented ones", () => {
    const source = "module github.com/jiva-studio/numen/a\n\nrequire github.com/jiva-studio/numen/b v0.0.0 // github.com/jiva-studio/numen/c\n";
    assert.deepEqual(getGoModReferences(source).imports, [{ line: 3, value: "github.com/jiva-studio/numen/b" }]);
  });
});

describe("the go workspace", () => {
  const work = "go 1.26\n\nuse (\n\t./libs/core\n\t./plugins/a\n)\n\nreplace x => ./libs/core\n";
  it("lists the used directories", () => {
    assert.deepEqual(parseWorkspaceUses(work), ["libs/core", "plugins/a"]);
    assert.deepEqual(parseWorkspaceUses("use ./single\n"), ["single"]);
    assert.deepEqual(parseWorkspaceUses("use ( // modules\n\t./a // first\n)\nuse ./b//c\n"), ["a", "b"]);
  });
  it("finds a module the workspace leaves out and a use without a module", () => {
    const gaps = findWorkspaceGaps(work, ["libs/core", "plugins/b"]).map((one) => one.symbol);
    assert.deepEqual(gaps, ["missing use ./plugins/b", "use ./plugins/a has no go.mod"]);
  });
  it("accepts a workspace that matches", () => {
    assert.deepEqual(findWorkspaceGaps(work, ["libs/core", "plugins/a"]), []);
  });
});

describe("the version of the linter", () => {
  const report = (go) => `golangci-lint has version 2.15.0 built with go${go} from (unknown) on (unknown)`;
  it("reads the Go version it was built with and the one the workspace declares", () => {
    assert.equal(parseBuiltWithGo(report("1.26.10")), "1.26.10");
    assert.equal(parseGoDirective("go 1.26.9\n\nuse (\n)\n"), "1.26.9");
  });
  it("compares versions by number", () => {
    assert.ok(compareGoVersions("1.26.10", "1.26.9") > 0);
    assert.ok(compareGoVersions("1.9", "1.26") < 0);
    assert.equal(compareGoVersions("1.26", "1.26.0"), 0);
  });
  it("fails a linter built with a Go older than the workspace", () => {
    assert.match(getLinterBuildGap(report("1.25.4"), "1.26.9"), /older than go1\.26\.9/);
    assert.match(getLinterBuildGap("golangci-lint has version 2.15.0", "1.26.9"), /does not report/);
  });
  it("accepts a linter built with the same or a newer Go", () => {
    assert.equal(getLinterBuildGap(report("1.26.9"), "1.26.9"), "");
    assert.equal(getLinterBuildGap(report("1.27.0"), "1.26.9"), "");
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

