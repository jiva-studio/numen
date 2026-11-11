import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { ESLint } from "eslint";
import { REPO_ROOT } from "./helpers.mjs";

/** The files a plugin copies from the golden template. */
const TEMPLATE = path.join(REPO_ROOT, "plugins/.example");

/** The lines of the messages of `ruleId` in the result of `file`. */
function findLines(results, file, ruleId) {
  const result = results.find((one) => one.filePath.endsWith(file));
  return [...new Set(result.messages.filter((one) => one.ruleId === ruleId).map((one) => one.line))];
}

/**
 * Lints planted files in a copy of the golden template. The copy lies inside the repository, in a folder whose name
 * starts with a dot, so the configuration resolves its packages the way a plugin does and no check reads it as a plugin.
 */
async function lintPlanted(files) {
  const dir = fs.mkdtempSync(path.join(REPO_ROOT, "plugins", ".golden-self-test-"));
  try {
    for (const name of ["eslint.config.js", "numen-rules.mjs", "tsconfig.json"]) {
      fs.copyFileSync(path.join(TEMPLATE, name), path.join(dir, name));
    }
    for (const [name, text] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
      fs.writeFileSync(path.join(dir, name), text);
    }
    return await new ESLint({ cwd: dir }).lintFiles(Object.keys(files));
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

describe("the golden plugin configuration", () => {
  it("fails a lib file that imports a ui, model or api segment at any depth, or a component", async () => {
    const results = await lintPlanted({
      "frontend/deck/lib/cards.ts": [
        "import { cardSize } from '../../ui/cards'",
        "import { kinds } from '../model/kinds'",
        "import type { Preset } from '../../api/types.js'",
        "import { Button } from '@numen/ui'",
        "import { order } from './order'",
        "import { day } from '../../lib/day'",
        "import type { Deck } from '../types'",
        "import Badge from './Badge.vue'",
        "import { stats } from './api/stats'",
        "import { deep } from '../../../deck/x/model/deep'",
        "import { tab } from '../../../../ui/tab'",
        "import { kind } from '../../../lib/kind'",
        "export const all = [cardSize, kinds, Button, order, day, Badge, stats, deep, tab, kind]",
        "export type Both = Preset | Deck",
        "",
      ].join("\n"),
    });
    assert.deepEqual(findLines(results, "frontend/deck/lib/cards.ts", "no-restricted-imports"), [1, 2, 3, 8, 9, 10, 11]);
  });

  it("fails a lib file that reaches the browser, a timer, the clock, a DOM type or the DOM", async () => {
    const results = await lintPlanted({
      "frontend/deck/lib/browser.ts": [
        "export const agent = navigator.userAgent",
        "export const later = (run: () => void) => setTimeout(run, 1)",
        "export const stamp = () => Date.now()",
        "export const read = (element: HTMLElement) => element.id",
        "export const press = (event: KeyboardEvent) => event.key",
        "export const build = (host: { createElement: (tag: string) => object }) => host.createElement('div')",
        "export const ok = (press: { readonly key: string }) => press.key",
        "",
      ].join("\n"),
    });
    assert.deepEqual(findLines(results, "frontend/deck/lib/browser.ts", "numen/pure-segment"), [1, 2, 3, 4, 5, 6]);
  });

  it("applies the component rules of the base config to a plugin component", async () => {
    const props = Array.from({ length: 8 }, (_, i) => `  field${i}: string`).join("\n");
    const results = await lintPlanted({
      "frontend/ui/Wide.vue": `<script setup lang="ts">\ndefineProps<{\n${props}\n}>()\n</script>\n\n<template>\n  <div />\n</template>\n`,
      "frontend/ui/Small.vue": `<script setup lang="ts">\ndefineProps<{ title: string }>()\n</script>\n\n<template>\n  <div />\n</template>\n`,
    });
    assert.deepEqual(findLines(results, "frontend/ui/Wide.vue", "numen/max-inline-props"), [2]);
    assert.deepEqual(findLines(results, "frontend/ui/Small.vue", "numen/max-inline-props"), []);
    const resolved = await new ESLint({ cwd: path.join(REPO_ROOT, "plugins/views-title-bar") }).calculateConfigForFile("frontend/ui/TitleBar.vue");
    for (const rule of ["numen/max-inline-props", "numen/vue-script-sections", "numen/max-script-lines"]) {
      assert.ok(resolved.rules[rule], `${rule} does not reach a plugin component`);
    }
  });
});
