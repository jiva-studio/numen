import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findExcludedSources, isTsConfig, readExcludeEntries } from "./check.mjs";

/** The findings for one configuration at `file` with `text`, over the tracked files. */
function findFor(file, text, tracked) {
  return findExcludedSources([{ file, entries: readExcludeEntries(file, text) }], tracked).map(({ line, symbol }) => `${line} ${symbol}`);
}

const TRACKED = [
  "apps/site/tsconfig.json",
  "apps/site/src/components/Demo.vue",
  "apps/site/src/pages/index.ts",
  "apps/site/scripts/build.ts",
  "apps/other/src/kept.ts",
];

describe("the typecheck scope rule", () => {
  it("passes an exclude of build output, which is not tracked", () => {
    const text = '{\n  "exclude": ["dist", "node_modules", "./coverage/"]\n}';
    assert.deepEqual(findFor("apps/site/tsconfig.json", text, TRACKED), []);
  });

  it("fails an excluded source file, folder or glob, at the line of the entry", () => {
    const text = [
      "{",
      '  "include": ["**/*.ts"],',
      '  "exclude": [',
      '    "dist",',
      '    "src/components/Demo.vue",',
      '    "scripts/",',
      '    "src/**/index.ts"',
      "  ]",
      "}",
    ].join("\n");
    assert.deepEqual(findFor("apps/site/tsconfig.json", text, TRACKED), [
      '5 exclude "src/components/Demo.vue" leaves out apps/site/src/components/Demo.vue',
      '6 exclude "scripts/" leaves out apps/site/scripts/build.ts',
      '7 exclude "src/**/index.ts" leaves out apps/site/src/pages/index.ts',
    ]);
  });

  it("reads an exclude that leaves the package for another tracked folder", () => {
    const text = '{ "exclude": ["../other/src"] }';
    assert.deepEqual(findFor("apps/site/tsconfig.json", text, TRACKED), ['1 exclude "../other/src" leaves out apps/other/src/kept.ts']);
  });

  it("reads a configuration with comments and no exclude", () => {
    assert.deepEqual(readExcludeEntries("tsconfig.json", '// base\n{ "compilerOptions": { "strict": true } }'), []);
  });

  it("recognises every TypeScript configuration by its name", () => {
    assert.deepEqual(
      ["tsconfig.json", "a/tsconfig.app.json", "tsconfig.base.json", "a/jsconfig.json", "a/package.json"].filter(isTsConfig),
      ["tsconfig.json", "a/tsconfig.app.json", "tsconfig.base.json"],
    );
  });
});
