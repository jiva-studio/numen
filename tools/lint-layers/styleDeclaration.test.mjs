import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { after, describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import { createBaseConfig } from "./base-eslint.mjs";

const numen = createBaseConfig(import.meta.dirname, { useProjectService: false }).find((entry) => entry.plugins?.numen).plugins.numen;

const project = fs.mkdtempSync(path.join(os.tmpdir(), "lint-layers-style-"));
fs.writeFileSync(
  path.join(project, "tsconfig.json"),
  JSON.stringify({ compilerOptions: { lib: ["ES2022", "DOM"], strict: true, target: "ES2022", module: "ESNext", skipLibCheck: true }, include: ["*.ts"] }),
);

after(() => fs.rmSync(project, { recursive: true, force: true }));

/** The lines the rule reports in a planted file; the file is typed against the DOM library. */
function findReportedLines(source) {
  const file = path.join(project, "planted.ts");
  fs.writeFileSync(file, source);
  const config = [
    {
      files: ["**/*.ts"],
      plugins: { numen },
      languageOptions: { parser: tseslint.parser, parserOptions: { projectService: true, tsconfigRootDir: project } },
      rules: { "numen/no-style-declaration-read": "error" },
    },
  ];
  const messages = new Linter({ cwd: project }).verify(source, config, { filename: file });
  const fatal = messages.filter((message) => message.fatal);
  assert.deepEqual(fatal, []);
  return messages.map((message) => message.line);
}

describe("the style declaration rule", () => {
  it("fails a property read from a computed style", () => {
    assert.deepEqual(findReportedLines("declare const el: Element\nconst a = getComputedStyle(el).overflow\n"), [2]);
    assert.deepEqual(findReportedLines("declare const el: Element\nconst a = getComputedStyle(el).getPropertyValue('--a')\n"), [2]);
  });

  it("fails a property read from the inline style of an element", () => {
    assert.deepEqual(findReportedLines("declare const el: HTMLElement\nconst a = el.style.left\n"), [2]);
    assert.deepEqual(findReportedLines("declare const el: HTMLElement\nconst a = el.style.getPropertyValue('--a')\n"), [2]);
  });

  it("fails a declaration that is kept or passed on, under any variable name", () => {
    const source = [
      "declare const el: Element",
      "declare function use(value: unknown): void",
      "const look = getComputedStyle(el)",
      "use(getComputedStyle(el))",
      "use((el as HTMLElement).style)",
      "const read = (declared: CSSStyleDeclaration) => declared.color",
    ].join("\n");
    assert.deepEqual(findReportedLines(source), [3, 4, 5, 6]);
  });

  it("passes writing a style, which sets a scene and reads nothing", () => {
    const source = ["declare const el: HTMLElement", "el.style.left = '1px'", "el.style.setProperty('--a', '1')", "el.style.removeProperty('--a')"].join("\n");
    assert.deepEqual(findReportedLines(source), []);
  });

  it("passes what a user can see happen: attributes, boxes and state", () => {
    const source = [
      "declare const el: HTMLElement",
      "const a = el.dataset['theme']",
      "const b = el.getBoundingClientRect().left",
      "const c = el.getAttribute('aria-expanded')",
      "const d = el.scrollHeight > el.clientHeight",
    ].join("\n");
    assert.deepEqual(findReportedLines(source), []);
  });

  it("decides by type: an object with style-like names is not a declaration", () => {
    const source = ["const box = { style: { left: 1, overflow: 'hidden' } }", "const a = box.style.left", "const b = box.style.overflow"].join("\n");
    assert.deepEqual(findReportedLines(source), []);
  });
});
