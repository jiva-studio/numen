import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import { numenPlugin } from "./base-eslint.mjs";

/** The lines and messages the rule reports in the source. */
function findReports(source) {
  const config = [
    {
      files: ["**/*.ts"],
      languageOptions: { parser: tseslint.parser },
      plugins: { numen: numenPlugin },
      rules: { "numen/pure-segment": "error" },
    },
  ];
  return new Linter().verify(source, config, { filename: "src/lib/probe.ts" }).map((message) => `${message.line}`);
}

describe("the pure segment rule", () => {
  it("fails the window, the document, the navigator and storage", () => {
    assert.deepEqual(findReports("export const a = window.innerWidth\nexport const b = document.body\nexport const c = navigator.userAgent"), ["1", "2", "3"]);
    assert.deepEqual(findReports("export const a = localStorage.getItem('k')\nexport const b = location.href"), ["1", "2"]);
  });

  it("fails a global reached through globalThis", () => {
    assert.deepEqual(findReports("export const a = globalThis.navigator.userAgent\nexport const b = globalThis.innerWidth"), ["1", "2"]);
  });

  it("fails timers, frames, microtasks and observers, also named by type", () => {
    const source = [
      "export const a = setTimeout(run, 1)",
      "clearInterval(1)",
      "requestAnimationFrame(run)",
      "queueMicrotask(run)",
      "export type Handle = ReturnType<typeof setTimeout>",
      "export const b = new ResizeObserver(run)",
    ].join("\n");
    assert.deepEqual(findReports(source), ["1", "2", "3", "4", "5", "6"]);
  });

  it("fails the clock and randomness, and passes a date made from a value", () => {
    assert.deepEqual(findReports("export const a = Date.now()\nexport const b = new Date()\nexport const c = performance.now()\nexport const d = Math.random()"), ["1", "2", "3", "4"]);
    assert.deepEqual(findReports("export const a = (ms: number) => new Date(ms)"), []);
  });

  it("fails an identity drawn from crypto, also through globalThis", () => {
    const source = [
      "export const a = () => crypto.randomUUID()",
      "export const b = (bytes: Uint8Array) => crypto.getRandomValues(bytes)",
      "export const c = () => globalThis.crypto.randomUUID()",
      "export const d = typeof crypto.randomUUID",
    ].join("\n");
    assert.deepEqual(findReports(source), ["1", "2", "3", "4"]);
    const given = [
      "export const a = (crypto: { randomUUID: () => string }) => crypto.randomUUID()",
      "export const b = (bytes: Uint8Array) => bytes.length",
    ].join("\n");
    assert.deepEqual(findReports(given), []);
  });

  it("fails DOM types, however the element is named", () => {
    const source = [
      "export const a = (element: HTMLElement) => element.id",
      "export const b = (press: KeyboardEvent) => press.key",
      "export const c = (node: Node, svg: SVGPathElement) => [node, svg]",
      "export const d = (box: DOMRect, range: Range) => [box, range]",
      "export const e = (target: Document | Window | Element) => target",
    ].join("\n");
    assert.deepEqual(findReports(source), ["1", "2", "3", "3", "4", "4", "5", "5", "5"]);
  });

  it("passes a type of the same name that a file declares or imports", () => {
    assert.deepEqual(findReports("import type { Range } from '@codemirror/state'\nexport const a = (found: Range<number>) => found"), []);
    assert.deepEqual(findReports("interface Event { at: number }\nexport const a = (one: Event) => one.at"), []);
    assert.deepEqual(findReports("export const a = (window: { id: string }, document: string) => window.id + document"), []);
  });

  it("fails building, querying and listening to the DOM through any object", () => {
    const source = [
      "export const a = (host: { createElement: (tag: string) => object }) => host.createElement('div')",
      "export const b = (frame: { addEventListener: (name: string) => void }) => frame.addEventListener('input')",
      "export const c = (frame: { dataset: object }) => frame.dataset",
      "export const d = (frame: { querySelectorAll: (q: string) => object[] }) => frame.querySelectorAll('td')",
    ].join("\n");
    assert.deepEqual(findReports(source), ["1", "2", "3", "4"]);
  });

  it("fails Vue, Pinia and a component", () => {
    assert.deepEqual(findReports("import { ref } from 'vue'\nimport { defineStore } from 'pinia'\nimport Card from './Card.vue'\nimport { x } from '@vue/reactivity'\nexport const a = [ref, defineStore, Card, x]"), ["1", "2", "3", "4"]);
  });

  it("fails a platform port taken from @numen/ui", () => {
    assert.deepEqual(findReports("import { doc, getNow, clamp } from '@numen/ui'\nexport const a = [doc, getNow, clamp]"), ["1", "1"]);
  });

  it("passes plain values, URL, Intl and JSON", () => {
    const source = [
      "export const a = (href: string) => new URL(href).host",
      "export const b = new Intl.NumberFormat('en').format(1)",
      "export const c = (text: string) => JSON.parse(text)",
      "export const d = (press: { readonly key: string }) => press.key === 'Enter'",
    ].join("\n");
    assert.deepEqual(findReports(source), []);
  });
});
