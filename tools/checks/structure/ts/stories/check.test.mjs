import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { createSources, findComponentsWithoutStories, isEmitting, listComponentExports, readStoryFile } from "./check.mjs";

const ENTRY = "libs/ui/kit/src/index.ts";
const ROOT = "libs/ui/kit/src";

const SILENT = "<script setup lang=\"ts\">\ndefineProps<{ label: string }>()\n</script>\n<template><p /></template>\n";
const EMITTING = "<script setup lang=\"ts\">\nconst emit = defineEmits<{ (event: 'pick', id: string): void }>()\n</script>\n<template><p /></template>\n";
const MODELLED = "<script setup lang=\"ts\">\nconst model = defineModel<boolean>()\n</script>\n<template><p /></template>\n";

const PLAIN_STORY = (name) =>
  `import type { Meta } from '@storybook/vue3-vite'\nimport ${name} from './${name}.vue'\nconst meta = { component: ${name} } satisfies Meta<typeof ${name}>\nexport default meta\nexport const Idle = { render: () => ({ components: { ${name} } }) }\n`;
const PLAYED_STORY = (name) =>
  `import { expect, userEvent } from 'storybook/test'\nimport ${name} from './${name}.vue'\nconst meta = { component: ${name} }\nexport default meta\nexport const Picks = { render: () => ({ components: { ${name} } }), play: async ({ args }) => { await userEvent.click(document.body); await expect(args.onPick).toHaveBeenCalledWith('a') } }\n`;

const LIBRARY = {
  [ENTRY]:
    "export { Chip } from './ui/chip'\nexport { Picker } from './ui/picker'\nexport { Toggle } from './ui/toggle'\nexport type { ChipSize } from './ui/chip'\nexport { measure } from './lib/measure'\n",
  [`${ROOT}/ui/chip/index.ts`]: "export { default as Chip } from './Chip.vue'\nexport type { ChipSize } from './size'\n",
  [`${ROOT}/ui/chip/Chip.vue`]: SILENT,
  [`${ROOT}/ui/chip/Chip.stories.ts`]: PLAIN_STORY("Chip"),
  [`${ROOT}/ui/picker/index.ts`]: "import Picker from './Picker.vue'\nexport { Picker }\n",
  [`${ROOT}/ui/picker/Picker.vue`]: EMITTING,
  [`${ROOT}/ui/picker/Picker.stories.ts`]: PLAYED_STORY("Picker"),
  [`${ROOT}/ui/toggle/index.ts`]: "export * from './parts'\n",
  [`${ROOT}/ui/toggle/parts.ts`]: "export { default as Toggle } from './Toggle.vue'\n",
  [`${ROOT}/ui/toggle/Toggle.vue`]: MODELLED,
  [`${ROOT}/ui/toggle/Toggle.stories.ts`]: PLAYED_STORY("Toggle"),
  [`${ROOT}/lib/measure.ts`]: "export function measure() { return 1 }\n",
};

/** A library with a silent component, an emitting one and a modelled one, each in its own folder; `overrides` replace files, and `null` removes one. */
function createLibrary(overrides = {}) {
  return createSources(Object.entries({ ...LIBRARY, ...overrides }).filter(([, text]) => text !== null));
}

const names = (found) => found.map((one) => one.symbol);

describe("the components a library exports", () => {
  it("follows named re-exports, imports exported again and export star, and skips types and functions", () => {
    const exported = listComponentExports(createLibrary(), ENTRY, ROOT);
    assert.deepEqual(
      [...exported],
      [
        ["Chip", `${ROOT}/ui/chip/Chip.vue`],
        ["Picker", `${ROOT}/ui/picker/Picker.vue`],
        ["Toggle", `${ROOT}/ui/toggle/Toggle.vue`],
      ],
    );
  });

  it("reads an export under another name by the name it is exported as", () => {
    const sources = createLibrary({ [ENTRY]: "export { Chip as Badge } from './ui/chip'\n" });
    assert.deepEqual([...listComponentExports(sources, ENTRY, ROOT).keys()], ["Badge"]);
  });

  it("ends on a cycle of re-exports", () => {
    const sources = createSources([
      [ENTRY, "export * from './a'\n"],
      [`${ROOT}/a.ts`, "export * from './b'\n"],
      [`${ROOT}/b.ts`, "export * from './a'\n"],
    ]);
    assert.equal(listComponentExports(sources, ENTRY, ROOT).size, 0);
  });
});

describe("the events of a component", () => {
  it("are declared by defineEmits or defineModel and by nothing else", () => {
    assert.equal(isEmitting(EMITTING), true);
    assert.equal(isEmitting(MODELLED), true);
    assert.equal(isEmitting(SILENT), false);
  });
});

describe("a story file", () => {
  it("renders the components it imports and uses, and not the ones it only imports", () => {
    const sources = createLibrary({
      [`${ROOT}/ui/chip/Chip.stories.ts`]: "import Chip from './Chip.vue'\nimport Picker from '../picker/Picker.vue'\nexport default { component: Chip }\n",
    });
    const read = readStoryFile(sources, `${ROOT}/ui/chip/Chip.stories.ts`, ROOT);
    assert.deepEqual([...read.rendered], [`${ROOT}/ui/chip/Chip.vue`]);
  });

  it("reaches a component through a barrel and through the alias of the source root", () => {
    const sources = createLibrary({
      [`${ROOT}/screens/Board.stories.ts`]:
        "import { Chip } from '@/ui/chip'\nimport { Picker as Chooser } from '../ui/picker'\nexport const A = { render: () => ({ components: { Chip, Chooser }, template: '<Chip /><chooser />' }) }\n",
    });
    const read = readStoryFile(sources, `${ROOT}/screens/Board.stories.ts`, ROOT);
    assert.deepEqual([...read.rendered], [`${ROOT}/ui/chip/Chip.vue`, `${ROOT}/ui/picker/Picker.vue`]);
  });

  it("has a play function when an exported story has one or the default export has one", () => {
    const sources = createLibrary({
      [`${ROOT}/a.stories.ts`]: "export default { play: async () => {} }\nexport const A = {}\n",
      [`${ROOT}/b.stories.ts`]: "const meta = { play() {} }\nexport default meta\nexport const A = {}\n",
      [`${ROOT}/c.stories.ts`]: "export default {}\nexport const A: Story = { play: async () => {} } satisfies Story\n",
      [`${ROOT}/d.stories.ts`]: "export default {}\nconst Hidden = { play: async () => {} }\nexport const A = {}\n",
      [`${ROOT}/e.stories.ts`]: "export default {}\nexport const A = { args: { play: 1 }, render: () => ({}) }\n",
    });
    const has = (name) => readStoryFile(sources, `${ROOT}/${name}.stories.ts`, ROOT).hasPlay;
    assert.deepEqual(["a", "b", "c", "d", "e"].map(has), [true, true, true, false, false]);
  });
});

describe("what a story draws", () => {
  const read = (text) => readStoryFile(createLibrary({ [`${ROOT}/x.stories.ts`]: text }), `${ROOT}/x.stories.ts`, ROOT);
  const CHIP = "import Chip from './ui/chip/Chip.vue'\n";

  it("is the component: of the story or of the default export, or the render target", () => {
    assert.deepEqual(read(CHIP + "export default {}\nexport const A = { component: Chip }\n").rendered.size, 1);
    assert.deepEqual(read(CHIP + "export default { component: Chip }\nexport const A = {}\n").stories[0].drawsComponent, true);
    assert.deepEqual(read(CHIP + "export default {}\nexport const A = { render: () => ({ components: { Chip }, template: '<Chip />' }) }\n").stories[0].drawsComponent, true);
    assert.deepEqual(read(CHIP + "import { h } from 'vue'\nexport default {}\nexport const A = { render: () => h(Chip) }\n").stories[0].drawsComponent, true);
    assert.deepEqual(read(CHIP + "const row = () => ({ components: { Chip }, template: '<Chip />' })\nexport default {}\nexport const A = { render: row }\n").stories[0].drawsComponent, true);
  });

  it("is not a component that is only imported, mentioned in a type, registered without a tag or listed", () => {
    for (const body of [
      "export default {}\nexport const A = { render: () => ({}) }\n",
      "export default {} satisfies Meta<typeof Chip>\nexport const A = {}\n",
      "export default {}\nexport const A = { render: () => ({ components: { Chip }, template: '<p />' }) }\n",
      "export default {}\nexport const A = { args: { parts: [Chip] } }\n",
    ]) {
      const found = read(CHIP + body);
      assert.deepEqual([found.rendered.size, found.stories[0].drawsComponent], [0, false], body);
    }
  });
});

describe("the story checks over a library", () => {
  it("fail a story that draws no component, and a play function that asserts nothing", () => {
    const sources = createLibrary({
      [`${ROOT}/ui/chip/Chip.stories.ts`]: PLAIN_STORY("Chip") + "export const Bare = { render: () => ({}), play: async () => {} }\n",
    });
    const found = findComponentsWithoutStories(sources, ENTRY);
    assert.deepEqual(found.bareStories.map((one) => one.symbol.split(" draws")[0]), []);
    assert.deepEqual(names(found.emptyPlays), ["story Bare has a play function with no expect"]);
  });

  it("fail a story file whose meta names no component and whose story draws none", () => {
    const sources = createLibrary({ [`${ROOT}/ui/chip/Chip.stories.ts`]: PLAIN_STORY("Chip").replace("component: Chip", "title: 'Chip'") });
    const found = findComponentsWithoutStories(sources, ENTRY);
    assert.deepEqual(found.bareStories.map((one) => one.symbol.split(" draws")[0]), ["story Idle"]);
    assert.deepEqual(names(found.unstoried), ["public component Chip is rendered by no story file"]);
  });

  it("fail an emitting component whose play asserts only the page, and pass one that asserts a spy or an emitted payload", () => {
    const stories = (play) =>
      `import Picker from './Picker.vue'\nimport { expect, fn, userEvent } from 'storybook/test'\nconst onPick = fn()\nexport default { component: Picker }\nexport const A = { args: { onPick }, play: ${play} }\n`;
    const run = (play) => names(findComponentsWithoutStories(createLibrary({ [`${ROOT}/ui/picker/Picker.stories.ts`]: stories(play) }), ENTRY).unplayed);
    const refused = ["public component Picker declares events and the play function of no story that draws it asserts an emitted event or a spy call after a user action"];
    assert.deepEqual(run("async () => {}"), refused);
    assert.deepEqual(run("async ({ canvasElement }) => { await expect(canvasElement.textContent).toBe('a') }"), refused);
    assert.deepEqual(run("async () => { await userEvent.click(document.body); await expect(onPick).toHaveBeenCalledOnce() }"), []);
    assert.deepEqual(run("async ({ args }) => { await userEvent.click(document.body); const { onPick: spy } = args; await expect(spy.mock.calls[0]).toEqual(['a']) }"), []);
    assert.deepEqual(run("async () => { await userEvent.click(document.body); await expect(emitted().pick).toEqual([['a']]) }"), []);
    assert.deepEqual(run("async () => { await fireEvent.keyDown(document.body); await expect(onPick).toHaveBeenCalledOnce() }"), []);
  });

  it("fail an emitting component whose play asserts a spy before any user action, or after none", () => {
    const stories = (play) =>
      `import Picker from './Picker.vue'\nimport { expect, fn, userEvent } from 'storybook/test'\nconst onPick = fn()\nexport default { component: Picker }\nexport const A = { args: { onPick }, play: ${play} }\n`;
    const run = (play) => names(findComponentsWithoutStories(createLibrary({ [`${ROOT}/ui/picker/Picker.stories.ts`]: stories(play) }), ENTRY).unplayed);
    const refused = ["public component Picker declares events and the play function of no story that draws it asserts an emitted event or a spy call after a user action"];
    assert.deepEqual(run("async () => { await expect(onPick).not.toHaveBeenCalled() }"), refused);
    assert.deepEqual(run("async () => { await expect(onPick).not.toHaveBeenCalled(); await userEvent.click(document.body) }"), refused);
  });

  it("pass when every component has a story and every emitting one has a play function", () => {
    const found = findComponentsWithoutStories(createLibrary(), ENTRY);
    assert.deepEqual(names(found.unstoried), []);
    assert.deepEqual(names(found.unplayed), []);
    assert.deepEqual([found.components, found.stories, found.emitting], [3, 3, 2]);
  });

  it("fail a component whose story file imports it and does not use it", () => {
    const sources = createLibrary({ [`${ROOT}/ui/chip/Chip.stories.ts`]: "import Chip from './Chip.vue'\nexport default {}\n" });
    assert.deepEqual(names(findComponentsWithoutStories(sources, ENTRY).unstoried), ["public component Chip is rendered by no story file"]);
  });

  it("fail a component with no story file at all", () => {
    const sources = createLibrary({ [`${ROOT}/ui/chip/Chip.stories.ts`]: null });
    assert.deepEqual(names(findComponentsWithoutStories(sources, ENTRY).unstoried), ["public component Chip is rendered by no story file"]);
  });

  it("fail an emitting component whose stories have no play function, and not a silent one", () => {
    const sources = createLibrary({
      [`${ROOT}/ui/picker/Picker.stories.ts`]: PLAIN_STORY("Picker"),
      [`${ROOT}/ui/chip/Chip.stories.ts`]: PLAIN_STORY("Chip"),
    });
    assert.deepEqual(names(findComponentsWithoutStories(sources, ENTRY).unplayed), [
      "public component Picker declares events and the play function of no story that draws it asserts an emitted event or a spy call after a user action",
    ]);
  });

  it("do not count a component that is not exported from the entry", () => {
    const sources = createLibrary({
      [`${ROOT}/ui/hidden/Hidden.vue`]: EMITTING,
      [`${ROOT}/ui/hidden/index.ts`]: "export { default as Hidden } from './Hidden.vue'\n",
    });
    const found = findComponentsWithoutStories(sources, ENTRY);
    assert.deepEqual([names(found.unstoried), names(found.unplayed), found.components], [[], [], 3]);
  });
});
