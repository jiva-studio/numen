import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  findWordsInGo,
  findWordsInScript,
  loadManifestWords,
  loadPluginDictionary,
  subtractDataWords,
} from "./checks/dictionary.mjs";
import {
  findDuplicateFiles,
  findImportCycles,
  findPluginCrossings,
  findSuppressions,
  findUnassignedPaths,
  findWorkspaceGaps,
  isGenerated,
  parseWorkspaceUses,
} from "./checks/discipline.mjs";
import { requireMatches } from "./checks/helpers.mjs";
import { compareGoVersions, getLinterBuildGap, parseBuiltWithGo, parseGoDirective } from "./checks/tools.mjs";

/** Manifests of two imaginary plugins, in the shape the real ones have. */
const MANIFESTS = {
  "plugins/alpha-view/manifest.json": JSON.stringify({
    id: "md.example.views.zephyr.graph",
    name: "Zephyr Graph",
    permissions: ["vault:read"],
    targets: ["desktop.editor"],
    settings: [{ key: "quokka_limit", type: "number", icon: "waypoints" }],
    contentTypes: ["quokkaformat"],
  }),
  "plugins/beta-view/manifest.json": JSON.stringify({
    id: "md.example.views.mango.player",
    name: "Mango Player",
    permissions: ["vault:read"],
    targets: ["desktop.editor"],
  }),
};

const readFixture = (file) => MANIFESTS[file];
const fixtureWords = () => loadManifestWords(Object.keys(MANIFESTS), readFixture);

describe("the dictionary of plugin words", () => {
  it("holds the ids, segments and values of the manifests", () => {
    const words = fixtureWords();
    for (const word of ["zephyr", "mango", "quokkaformat", "md.example.views.zephyr.graph"]) assert.ok(words.has(word), word);
  });

  it("holds no segment that every plugin id shares, no presentation value and no host target", () => {
    const words = fixtureWords();
    for (const word of ["md", "example", "waypoints", "quokka_limit", "desktop.editor", "editor"]) assert.equal(words.has(word), false, word);
  });

  it("drops the words an SDK or the protocol defines", () => {
    const dictionary = subtractDataWords(fixtureWords(), new Set(["vault", "read", "mango", "player"]));
    assert.equal(dictionary.has("mango"), false);
    assert.ok(dictionary.has("zephyr"));
  });

  it("is built from the live manifests, enough to examine", () => {
    assert.ok(loadPluginDictionary().size >= 10);
  });
});

describe("a plugin word in a frontend source", () => {
  const dictionary = subtractDataWords(fixtureWords(), new Set());
  const cases = {
    "a string literal": "const kind = 'zephyr'",
    "a template literal": "const kind = `zephyr`",
    "an object key": "const table = { zephyr: 1 }",
    "an enum member": "enum Kind { Zephyr }",
    "a case label": "switch (kind) { case 'zephyr': break }",
    "an identifier": "const zephyr = 1",
    "a segment of a literal": "const filter = 'feature:zephyr'",
    "a word of a camel-case identifier": "const getZephyrGraph = () => 1",
  };
  for (const [name, text] of Object.entries(cases)) {
    it(`is found as ${name}`, () => {
      assert.ok(findWordsInScript("fixture.ts", dictionary, text).length > 0, text);
    });
  }

  it("is found in the script block of a component", () => {
    const text = "<template><p /></template>\n<script setup lang=\"ts\">\nconst a = 'mango'\n</script>\n";
    const found = findWordsInScript("Fixture.vue", dictionary, text);
    assert.equal(found.length, 1);
    assert.equal(found[0].line, 3);
  });

  it("is not found in an import path or in a word that only contains it", () => {
    assert.deepEqual(findWordsInScript("fixture.ts", dictionary, "import x from './zephyr'\nconst a = 'zephyrs'"), []);
  });

  it("is found for a word of any live manifest, so no name is listed in the check", () => {
    const live = loadPluginDictionary();
    const word = [...live.keys()].find((one) => /^[a-z]+$/.test(one));
    assert.ok(word, "the live dictionary holds a plain word");
    assert.ok(findWordsInScript("fixture.ts", live, `const kind = '${word}'`).length > 0);
  });

  it("is found in a Go source as a literal and as an identifier", () => {
    const found = findWordsInGo("fixture.go", dictionary, 'package a\nvar mango = "zephyr"\n');
    assert.equal(found.length, 2);
  });
});

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
    { file: "plugins/alpha/frontend/a.ts", text: "import { x } from './b'" },
    { file: "plugins/beta/frontend/b.ts", text: "export const x = 1" },
  ];
  it("accepts references to the own plugin", () => {
    assert.deepEqual(findPluginCrossings(base, packages), []);
  });
  it("finds a Go import of another plugin", () => {
    const sources = [...base, { file: "plugins/alpha/backend/a.go", text: 'import "github.com/jiva-studio/numen/plugins/beta/backend"' }];
    assert.equal(findPluginCrossings(sources, packages).length, 1);
  });
  it("finds a relative import that resolves into another plugin", () => {
    const sources = [...base, { file: "plugins/alpha/frontend/c.ts", text: "import { x } from '../../beta/frontend/b'" }];
    assert.equal(findPluginCrossings(sources, packages).length, 1);
  });
  it("finds the package name of another plugin and a dependency on it", () => {
    const sources = [
      ...base,
      { file: "plugins/alpha/frontend/c.ts", text: "import { x } from '@example/beta/lib'" },
      { file: "plugins/alpha/package.json", text: JSON.stringify({ dependencies: { "@example/beta": "1.0.0" } }) },
    ];
    assert.equal(findPluginCrossings(sources, packages).length, 2);
  });
  it("finds a path or alias that names another plugin", () => {
    const sources = [...base, { file: "plugins/alpha/vite.config.ts", text: "const a = '../beta/x'; const b = '@plugins/beta'; const c = '../../plugins/beta/y'" }];
    assert.equal(findPluginCrossings(sources, packages).length, 2);
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
  it("finds a file named in an eslint ignores or files entry", () => {
    assert.equal(found("eslint.config.js", "export default [{ ignores: ['**/clock.ts', 'dist/**'] }]").length, 1);
    assert.equal(found("eslint.config.js", "export default [{ files: ['src/app/thing.ts'] }]").length, 1);
    assert.equal(found("eslint.config.js", "export default [{ files: ['**/*.ts'], ignores: ['dist/**'] }]").length, 0);
  });
  it("finds a list of repository files in a check", () => {
    assert.equal(found("tools/lint-layers/checks/x.mjs", 'const EXEMPT = ["apps/desktop/a.ts", "libs/x/b.go"]').length, 1);
    assert.equal(found("tools/lint-layers/checks/x.mjs", 'const ROOTS = ["apps/desktop", "libs/x"]').length, 0);
  });
  it("does not read a marker inside a string, or generated code", () => {
    assert.deepEqual(found("a.ts", "const a = '// eslint-disable-next-line'"), []);
    assert.deepEqual(found("a.go", "// Code generated by protoc. DO NOT EDIT.\n\npackage a\n//nolint\n"), []);
    assert.ok(isGenerated("// Code generated by x. DO NOT EDIT.\npackage a"));
    assert.equal(isGenerated("package a\n// Code generated by x. DO NOT EDIT."), true);
  });
});

describe("the duplicate files of plugins", () => {
  const body = Array.from({ length: 30 }, (_, index) => `export const value${index} = compute(${index}, 'name${index}')`).join("\n");
  const copy = (plugin, name, text) => ({ file: `plugins/${plugin}/frontend/${name}`, text });
  it("finds an identical file under another name", () => {
    const found = findDuplicateFiles([copy("a", "one.ts", body), copy("b", "two.ts", body)]);
    assert.equal(found.length, 1);
  });
  it("finds a copy that drifted, with a line added and the formatting changed", () => {
    const drifted = body.replace("value3 = compute(3, 'name3')", "value3 =\n  compute(3, 'name3')") + "\nexport const extra = 1";
    assert.equal(findDuplicateFiles([copy("a", "dom.ts", body), copy("b", "dom.ts", drifted)]).length, 1);
  });
  it("finds a copy in several plugins", () => {
    const sources = ["a", "b", "c"].map((plugin) => copy(plugin, "dom.ts", body));
    assert.equal(findDuplicateFiles(sources).length, 3);
  });
  it("accepts different files of one name and the same file twice in one plugin", () => {
    const other = Array.from({ length: 30 }, (_, index) => `function other${index}(): string { return 'x${index}' }`).join("\n");
    assert.equal(findDuplicateFiles([copy("a", "dom.ts", body), copy("b", "dom.ts", other)]).length, 0);
    assert.equal(findDuplicateFiles([copy("a", "one.ts", body), copy("a", "two.ts", body)]).length, 0);
  });
  it("ignores generated files and files too small to be a copy", () => {
    assert.equal(findDuplicateFiles([copy("a", "g.ts", "// @generated\n" + body), copy("b", "g.ts", "// @generated\n" + body)]).length, 0);
    assert.equal(findDuplicateFiles([copy("a", "s.ts", "export const a = 1"), copy("b", "s.ts", "export const a = 1")]).length, 0);
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

describe("the go workspace", () => {
  const work = "go 1.26\n\nuse (\n\t./libs/core\n\t./plugins/a\n)\n\nreplace x => ./libs/core\n";
  it("lists the used directories", () => {
    assert.deepEqual(parseWorkspaceUses(work), ["libs/core", "plugins/a"]);
    assert.deepEqual(parseWorkspaceUses("use ./single\n"), ["single"]);
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

describe("the import cycles of frontend files", () => {
  const roots = new Set(["pkg"]);
  const source = (file, text) => ({ file: `pkg/src/${file}`, text });
  it("finds two files that import each other", () => {
    const found = findImportCycles([source("a.ts", "import { b } from './b'"), source("b.ts", "import { a } from './a'")], roots);
    assert.equal(found.length, 1);
  });
  it("finds a cycle through the alias and an index file", () => {
    const sources = [source("a/index.ts", "export * from './a'"), source("a/a.ts", "import { x } from '@/c'"), source("c.ts", "import { y } from '@/a'")];
    assert.equal(findImportCycles(sources, roots).length, 1);
  });
  it("finds a file that imports itself", () => {
    assert.equal(findImportCycles([source("a.ts", "import { a } from './a'")], roots).length, 1);
  });
  it("accepts an acyclic graph and a cycle that only types close", () => {
    assert.equal(findImportCycles([source("a.ts", "import { b } from './b'"), source("b.ts", "export const b = 1")], roots).length, 0);
    const typed = [source("a.ts", "import type { B } from './b'"), source("b.ts", "import { a } from './a'")];
    assert.equal(findImportCycles(typed, roots).length, 0);
  });
});
