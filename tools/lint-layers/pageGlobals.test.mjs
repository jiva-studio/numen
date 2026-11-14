import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import { createBaseConfig, createPortConfig } from "./base-eslint.mjs";

const BANS = new Set(["no-restricted-globals", "no-restricted-properties"]);

// The order of the editor: the base, then the port block over every source file.
const config = [
  ...createBaseConfig(import.meta.dirname, { useProjectService: false }),
  createPortConfig({ files: ["src/**/*.ts"], ignores: ["src/**/*.test.ts"] }),
];

/** The lines the global bans report in the source of the file. */
function findBannedLines(source, filename) {
  return new Linter()
    .verify(source, config, { filename })
    .filter((message) => BANS.has(message.ruleId))
    .map((message) => message.line);
}

describe("the page globals in model/ after the port block", () => {
  const model = "src/features/note-commands/model/commands.ts";

  it("fail the navigator, the window, the document and the location, bare", () => {
    const source = [
      "export const a = navigator.clipboard",
      "export const b = window.innerWidth",
      "export const c = document.body",
      "export const d = location.search",
    ].join("\n");
    assert.deepEqual(findBannedLines(source, model), [1, 2, 3, 4]);
  });

  it("fail the same globals through globalThis", () => {
    const source = [
      "export const a = globalThis.location.search",
      "export const b = globalThis.navigator.userAgent",
      "export const c = globalThis.window",
      "export const d = globalThis.document",
    ].join("\n");
    assert.deepEqual(findBannedLines(source, model), [1, 2, 3, 4]);
  });

  it("keep the lifetimes banned", () => {
    assert.deepEqual(findBannedLines("requestAnimationFrame(() => 0)", model), [1]);
  });

  it("pass a page global taken as a parameter", () => {
    const source = "export const read = (page: { navigator: { userAgent: string } }) => page.navigator.userAgent";
    assert.deepEqual(findBannedLines(source, model), []);
  });
});

describe("the page globals outside model/", () => {
  it("pass in another segment, where the lifetimes stay banned", () => {
    const source = ["export const a = navigator.userAgent", "requestAnimationFrame(() => 0)"].join("\n");
    assert.deepEqual(findBannedLines(source, "src/app/window/getUserAgent.ts"), [2]);
  });

  it("pass in a test of a model", () => {
    assert.deepEqual(findBannedLines("export const a = globalThis.location.search", "src/features/a/model/a.test.ts"), []);
  });
});
