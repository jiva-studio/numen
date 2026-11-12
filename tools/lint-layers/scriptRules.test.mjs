import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { Linter } from "eslint";
import tseslint from "typescript-eslint";
import vueParser from "vue-eslint-parser";
import { numenPlugin as numen } from "./base-eslint.mjs";

/** The message ids and lines a rule reports in a component. */
function findComponentReports(rule, script, template = "<div />") {
  const source = `<script setup lang="ts">\n${script}\n</script>\n\n<template>\n${template}\n</template>\n`;
  const config = [
    {
      files: ["**/*.vue"],
      plugins: { numen },
      languageOptions: { parser: vueParser, parserOptions: { parser: tseslint.parser, sourceType: "module" } },
      rules: { [`numen/${rule}`]: "error" },
    },
  ];
  return new Linter().verify(source, config, { filename: "src/Part.vue" }).map((message) => `${message.line} ${message.messageId ?? message.message}`);
}

describe("the script order rule", () => {
  it("passes statements in the order of props, state, hooks, handlers, helpers, with or without banners", () => {
    const script = [
      "import { ref, watch } from 'vue'",
      "const props = defineProps<{ a: string }>()",
      "const emit = defineEmits<{ done: [] }>()",
      "const count = ref(0)",
      "watch(count, () => emit('done'))",
      "function onSave() { count.value++ }",
      "const format = (n: number) => String(n)",
      "function pad(n: number) { return format(n) + props.a }",
    ].join("\n");
    assert.deepEqual(findComponentReports("vue-script-sections", script), []);
  });

  it("fails a statement that comes after one of a later section, whatever comments it carries", () => {
    const script = [
      "import { ref, onMounted } from 'vue'",
      "function onSave() { return 1 }",
      "const count = ref(0)",
      "onMounted(() => count.value++)",
      "const props = defineProps<{ a: string }>()",
      "function format() { return 1 }",
      "const other = ref(1)",
    ].join("\n");
    assert.deepEqual(findComponentReports("vue-script-sections", script), ["4 outOfOrder", "5 outOfOrder", "6 outOfOrder", "8 outOfOrder"]);
  });

  it("does not demand a banner of a long script", () => {
    const script = Array.from({ length: 60 }, (_, i) => `const value${i} = ref(${i})`).join("\n");
    assert.deepEqual(findComponentReports("vue-script-sections", `import { ref } from 'vue'\n${script}`), []);
  });
});

describe("the handler naming rule", () => {
  it("fails a function the template binds to an event unless it is named on<Action>", () => {
    const script = ["function toggle() { return 1 }", "function onClose() { return 2 }", "const select = () => 3"].join("\n");
    const template = '  <button @click="toggle" @keydown="onClose" @input="select($event)">x</button>';
    assert.deepEqual(findComponentReports("handler-naming", script, template), ["2 invalidHandlerName", "4 invalidHandlerName"]);
  });

  it("fails a handle prefix on any function of the script, bound or not", () => {
    const script = "function handleClick() { return 1 }\nconst handleOther = () => 2";
    assert.deepEqual(findComponentReports("handler-naming", script), ["2 forbiddenHandlePrefix", "3 forbiddenHandlePrefix"]);
  });

  it("passes a helper the template calls in an expression and a function the template does not bind", () => {
    const script = "function format() { return 1 }";
    assert.deepEqual(findComponentReports("handler-naming", script, '  <p :title="format()" />'), []);
  });
});

/** The slice reports for a test file at a path. */
function findMockReports(source, filename) {
  const config = [{ files: ["**/*.ts"], plugins: { numen }, languageOptions: { parser: tseslint.parser }, rules: { "numen/no-mock-of-subject-slice": "error" } }];
  return new Linter({ cwd: "/repo" }).verify(source, config, { filename }).map((message) => `${message.line} ${message.messageId ?? message.message}`);
}

describe("the mock of the subject slice rule", () => {
  it("fails vi.mock of a module in the slice of the test, by relative path and by alias", () => {
    const file = "/repo/apps/w/src/entities/tab/model/open.test.ts";
    assert.deepEqual(findMockReports("vi.mock('../api/saved', () => ({}))", file), ["1 mocksOwnSlice"]);
    assert.deepEqual(findMockReports("vi.mock('@/entities/tab/lib/x', () => ({}))", file), ["1 mocksOwnSlice"]);
    assert.deepEqual(findMockReports("vi.doMock('./x', () => ({}))", file), ["1 mocksOwnSlice"]);
  });

  it("passes vi.mock of another slice, another layer and a package", () => {
    const file = "/repo/apps/w/src/entities/tab/model/open.test.ts";
    assert.deepEqual(findMockReports("vi.mock('@/entities/note', () => ({}))", file), []);
    assert.deepEqual(findMockReports("vi.mock('@/shared/notices/messages', () => ({}))", file), []);
    assert.deepEqual(findMockReports("vi.mock('@capacitor/core', () => ({}))", file), []);
  });

  it("reads the first folder under frontend as the slice of a plugin", () => {
    const file = "/repo/plugins/p/frontend/ui/Tab.test.ts";
    assert.deepEqual(findMockReports("vi.mock('./Other.vue', () => ({}))", file), ["1 mocksOwnSlice"]);
    assert.deepEqual(findMockReports("vi.mock('../model/useReader', () => ({}))", file), []);
  });

  it("fails vi.spyOn on a module or value from the slice of the test, and passes a spy on a fake port or another slice", () => {
    const file = "/repo/apps/w/src/app/window/windowSettings.test.ts";
    const violating = [
      "import { createWindowSettingsPart } from './windowSettings'",
      "const create = () => ({ settings: createWindowSettingsPart() })",
      "const { settings } = create()",
      "vi.spyOn(settings.rest, 'start')",
    ].join("\n");
    assert.deepEqual(findMockReports(violating, file), ["4 spiesOnOwnSlice"]);

    const passing = [
      "import { createTestTabs } from '@/testing/tabs'",
      "const windowTabs = createTestTabs()",
      "vi.spyOn(windowTabs.groups, 'closeTabsOfPlugin')",
    ].join("\n");
    assert.deepEqual(findMockReports(passing, file), []);
  });
});

/** The lines the style assertions of a test report. */
function findStyleReports(source) {
  const config = [{ files: ["**/*.ts"], plugins: { numen }, languageOptions: { parser: tseslint.parser }, rules: { "numen/no-style-declaration-read": "error" } }];
  return new Linter().verify(source, config, { filename: "a.test.ts" }).map((message) => message.line);
}

describe("the style assertion rule", () => {
  it("fails toHaveStyle, getAttribute('style') and expect of a style property", () => {
    const source = ["expect(el).toHaveStyle('a: b')", "const s = el.getAttribute('style')", "expect(props?.style).toBe(x)", "expect(el.getAttribute('class')).toBe('a')"].join("\n");
    assert.deepEqual(findStyleReports(source), [1, 2, 3]);
  });
});
