import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findFactoriesInScriptSetup, findLogicInApp, findPassThroughFactories, findWideAppFunctions } from "./check.mjs";

const APP = "apps/desktop/editor/src/app";

const file = (name, text) => ({ file: `${APP}/${name}`, text });
const symbols = (found) => found.findings.map((one) => `${one.file.split("/").pop()}:${one.line} ${one.symbol}`);

describe("the app layer holds no logic", () => {
  it("fails a template literal and a concatenation that build text, once per text", () => {
    const found = findLogicInApp([
      file("a.ts", ["export const a = (n: number) => `Saved ${n} notes`", "export const b = (n: string) => 'Failed: ' + n + '.'"].join("\n")),
    ]);
    assert.deepEqual(symbols(found), ["a.ts:1 text built with a template literal", "a.ts:2 text built by concatenation"]);
  });

  it("fails an error text and passes an error that carries no text of its own", () => {
    const found = findLogicInApp([file("a.ts", ["throw new Error('no window')", "throw new Error(message)"].join("\n"))]);
    assert.deepEqual(symbols(found), ["a.ts:1 error text in Error"]);
  });

  it("fails an adapter function that calls and decides, in every form of the decision", () => {
    const text = [
      "export const adapter = {",
      "  a: (x) => { if (x) { run(x) } },",
      "  b: (x) => (x ? run(x) : undefined),",
      "  c: async (x) => { try { await run(x) } catch { fail() } },",
      "  d: (x) => run(x?.id ?? 'none'),",
      "  e(x) { return x.ok ? run(x) : fail() },",
      "}",
    ].join("\n");
    assert.deepEqual(symbols(findLogicInApp([file("a.ts", text)])), [
      "a.ts:2 adapter function a makes a call and decides",
      "a.ts:3 adapter function b makes a call and decides",
      "a.ts:4 adapter function c makes a call and decides",
      "a.ts:5 adapter function d makes a call and decides",
      "a.ts:6 adapter function e makes a call and decides",
    ]);
  });

  it("counts && , || , ?? and an array method with a callback as decisions, without an optional chain", () => {
    const text = [
      "export const adapter = {",
      "  a: (x) => run(x && x.id),",
      "  b: (x) => run(x || 'none'),",
      "  c: (x) => run(x ?? 'none'),",
      "  d: (xs) => run(xs.map((one) => one.id)),",
      "  e: (xs) => run(xs.filter(isOpen)),",
      "}",
    ].join("\n");
    assert.deepEqual(symbols(findLogicInApp([file("a.ts", text)])), [
      "a.ts:2 adapter function a makes a call and decides",
      "a.ts:3 adapter function b makes a call and decides",
      "a.ts:4 adapter function c makes a call and decides",
      "a.ts:5 adapter function d makes a call and decides",
    ]);
  });

  it("passes a function that only calls, only decides, or is no property of an object", () => {
    const text = [
      "export const adapter = {",
      "  travel: async (path) => { await navigation.travel(path) },",
      "  isOpen: (x) => (x ? true : false),",
      "  read: (x) => x?.id,",
      "}",
      "export function pick(x) { if (x) { return run(x) } return undefined }",
    ].join("\n");
    assert.deepEqual(symbols(findLogicInApp([file("a.ts", text)])), []);
  });

  it("reads the script of a component with its line in the file, and skips tests", () => {
    const vue = ["<template><div /></template>", "<script setup lang=\"ts\">", "const label = `n ${1}`", "</script>"].join("\n");
    assert.deepEqual(symbols(findLogicInApp([file("App.vue", vue), file("a.test.ts", "const a = `n ${1}`")])), ["App.vue:3 text built with a template literal"]);
  });

  it("counts the files it read", () => {
    assert.equal(findLogicInApp([file("a.ts", "export const a = 1"), file("a.test.ts", "")]).read, 1);
  });
});

describe("a script setup holds no factory and no async function", () => {
  const vue = (script) => ({ file: "libs/ui/a/src/A.vue", text: `<template><div /></template>\n<script setup lang="ts">\n${script}\n</script>\n` });

  it("fails a call of a create function, a method named create, and an async function", () => {
    const found = findFactoriesInScriptSetup([
      vue(["const tabs = createTabs()", "const x = parts.createRow()", "const load = async () => { await fetchAll() }", "async function save() {}"].join("\n")),
    ]);
    assert.deepEqual(symbols(found), [
      "A.vue:3 <script setup> calls createTabs",
      "A.vue:4 <script setup> calls createRow",
      "A.vue:5 <script setup> holds an async function",
      "A.vue:6 <script setup> holds an async function",
    ]);
  });

  it("passes DOM creators, other calls and a plain script block", () => {
    const found = findFactoriesInScriptSetup([
      vue("const el = document.createElement('div')\nconst s = useTabs()"),
      { file: "libs/ui/a/src/B.vue", text: "<script lang=\"ts\">\nexport const a = createTabs()\n</script>\n<script setup lang=\"ts\">\nconst b = 1\n</script>" },
    ]);
    assert.deepEqual(symbols(found), []);
    assert.equal(found.read, 2);
  });
});

describe("a create function is more than a pass-through", () => {
  const src = (text) => [{ file: "apps/desktop/editor/src/app/window/a.ts", text }];

  it("fails a factory that copies the fields of its parameter one by one", () => {
    const found = findPassThroughFactories(
      src(
        [
          "export function createServices(deps: Deps) { return { a: deps.a, b: deps.b } }",
          "export const createPart = ({ a, b }: Deps) => ({ a, b })",
          "export const createPair = (a: A, b: B) => ({ a, b: b })",
        ].join("\n"),
      ),
    );
    assert.deepEqual(symbols(found), [
      "a.ts:1 createServices only copies fields of its parameter",
      "a.ts:2 createPart only copies fields of its parameter",
      "a.ts:3 createPair only copies fields of its parameter",
    ]);
  });

  it("passes a factory that does other work, copies nothing, or is not named create", () => {
    const found = findPassThroughFactories(
      src(
        [
          "export function createA(deps: Deps) { const s = make(deps.a); return { a: s, b: deps.b } }",
          "export const createB = (deps: Deps) => ({ a: deps.a, b: make(deps.b) })",
          "export const createC = (deps: Deps) => ({ ...deps })",
          "export const createD = (deps: Deps) => ({ a: deps.a })",
          "export const wrapE = (deps: Deps) => ({ a: deps.a, b: deps.b })",
        ].join("\n"),
      ),
    );
    assert.deepEqual(symbols(found), []);
    assert.equal(found.read, 4);
  });
});

describe("an app function takes few inputs and narrows no result", () => {
  const src = (text) => [{ file: "apps/desktop/editor/src/app/window/a.ts", text }];

  it("fails a function with more than eight inputs, in a parameter list or in the object it takes", () => {
    const found = findWideAppFunctions(
      src(
        [
          "export const createWide = (deps: { a: A; b: B; c: C; d: D; e: E; f: F; g: G; h: H; i: I }) => deps",
          "export function createMany(a: A, b: B, c: C, d: D, e: E, f: F, g: G, h: H, i: I) { return a }",
          "export const createEight = (deps: { a: A; b: B; c: C; d: D; e: E; f: F; g: G; h: H }) => deps",
        ].join("\n"),
      ),
    );
    assert.deepEqual(symbols(found), ["a.ts:1 createWide takes 9 inputs, at most 8 are allowed", "a.ts:2 createMany takes 9 inputs, at most 8 are allowed"]);
  });

  it("fails an input that narrows the result of another function, directly or by an alias", () => {
    const found = findWideAppFunctions(
      src(
        [
          "export const createStage = () => ({ a: 1, b: 2 })",
          "type Stage = ReturnType<typeof createStage>",
          "export const createDirect = (deps: { stage: Pick<ReturnType<typeof createStage>, 'a'> }) => deps",
          "export const createAlias = (stage: Omit<Stage, 'b'>) => stage",
          "export interface Deps { readonly stage: Pick<Stage, 'a'> }",
          "export const createDeclared = (deps: Deps) => deps",
        ].join("\n"),
      ),
    );
    assert.deepEqual(symbols(found), [
      "a.ts:3 createDirect takes stage, a narrowed result of a function",
      "a.ts:4 createAlias takes stage, a narrowed result of a function",
      "a.ts:6 createDeclared takes stage, a narrowed result of a function",
    ]);
  });

  it("passes a result taken whole, a part of a result, a type of another function and few inputs", () => {
    const found = findWideAppFunctions(
      src(
        [
          "export const createStage = () => ({ a: 1, b: 2 })",
          "export const createWhole = (deps: { stage: ReturnType<typeof createStage> }) => deps",
          "export const createPart = (deps: { a: ReturnType<typeof createStage>['a']; b: B }) => deps",
          "export const createOther = (deps: { host: Pick<ReturnType<typeof useHost>, 'a'> }) => deps",
        ].join("\n"),
      ),
    );
    assert.deepEqual(symbols(found), []);
  });

  it("skips the tests", () => {
    const found = findWideAppFunctions([{ file: "apps/desktop/editor/src/app/window/a.test.ts", text: "export const f = (a: A, b: B, c: C, d: D, e: E, f: F, g: G, h: H, i: I) => a" }]);
    assert.equal(found.read, 0);
  });
});
