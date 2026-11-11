import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { APP_MAX_LINES, findAppLogic, findAppOverreach, findComponentsBelowUi, findCoreClientOutsideApi, findDeepSliceImports, findLateBindings, findPageImports, findSharedServingOneSlice, getSegment, getSliceOf } from "./frontendGraph.mjs";

/** A module of the resolved graph with local imports `{ specifier: resolvedPath }`. */
function moduleOf(source, imports = {}) {
  return {
    source,
    dependencies: Object.entries(imports).map(([module, resolved]) => ({ module, resolved, couldNotResolve: false })),
  };
}

const PLUGIN = "plugins/views-cards/frontend";

describe("the segment of a file", () => {
  it("reads the segment from the nearest segment folder", () => {
    assert.equal(getSegment(`${PLUGIN}/deck/lib/cards.ts`), "lib");
    assert.equal(getSegment(`${PLUGIN}/ui/lib/size.ts`), "lib");
    assert.equal(getSegment(`${PLUGIN}/ui/Cards.vue`), "ui");
    assert.equal(getSegment(`${PLUGIN}/index.ts`), undefined);
  });
});

describe("the component placement rule", () => {
  it("fails a component in a lib, api or model segment and passes one in ui or outside a segment", () => {
    const modules = [
      moduleOf(`${PLUGIN}/deck/lib/Badge.vue`),
      moduleOf(`${PLUGIN}/deck/model/Pane.vue`),
      moduleOf(`libs/ui/components/src/shared/api/Probe.vue`),
      moduleOf(`${PLUGIN}/deck/ui/Deck.vue`),
      moduleOf(`${PLUGIN}/deck/ui/Deck.vue`),
      moduleOf(`${PLUGIN}/DocumentTab.vue`),
      moduleOf(`${PLUGIN}/lib/size.ts`),
      moduleOf(`node_modules/x/model/A.vue`),
    ];
    const { findings, read } = findComponentsBelowUi(modules);
    assert.deepEqual(
      findings.map((one) => `${one.file} ${one.symbol}`),
      [
        `${PLUGIN}/deck/lib/Badge.vue a component in a lib segment`,
        `${PLUGIN}/deck/model/Pane.vue a component in a model segment`,
        `libs/ui/components/src/shared/api/Probe.vue a component in a api segment`,
      ],
    );
    assert.equal(read, 5);
  });
});

const WINDOW = "apps/desktop/editor/src";

describe("the core client rule of windows", () => {
  it("passes the entry and an api segment, and fails every other layer", () => {
    const modules = [
      moduleOf(`${WINDOW}/main.ts`, { "@numen/core-client": "libs/core/client/src/index.ts" }),
      moduleOf(`${WINDOW}/entities/note/api/notes.ts`, { "@numen/core-client": "libs/core/client/src/index.ts" }),
      moduleOf(`${WINDOW}/features/tree/model/tree.ts`, { "@numen/core-client": "libs/core/client/src/index.ts" }),
      moduleOf(`${WINDOW}/shared/lib/client.ts`, { "@numen/core-client/errors": "libs/core/client/src/errors.ts" }),
      moduleOf(`${WINDOW}/pages/home/ui/Home.vue`, { "@numen/ui": "libs/ui/components/src/index.ts" }),
      moduleOf(`${WINDOW}/app/app.test.ts`, { "@numen/core-client": "libs/core/client/src/index.ts" }),
    ];
    assert.deepEqual(findCoreClientOutsideApi(modules, [WINDOW]), [
      { file: `${WINDOW}/features/tree/model/tree.ts`, line: 1, symbol: "imports @numen/core-client from the features layer" },
      { file: `${WINDOW}/shared/lib/client.ts`, line: 1, symbol: "imports @numen/core-client/errors from the shared layer" },
    ]);
  });
});

describe("the shared layer rule of windows", () => {
  it("names the slice of a file on a sliced layer and the layer otherwise", () => {
    assert.equal(getSliceOf(`${WINDOW}/features/tree/model/a.ts`, WINDOW), "features/tree");
    assert.equal(getSliceOf(`${WINDOW}/app/App.vue`, WINDOW), "app");
    assert.equal(getSliceOf(`${WINDOW}/main.ts`, WINDOW), "main.ts");
  });

  it("fails a shared file one slice uses with or without the app layer, directly or through other shared files, and passes one two slices use", () => {
    const modules = [
      moduleOf(`${WINDOW}/features/tree/ui/Tree.vue`, {
        "@/shared/lib/icons": `${WINDOW}/shared/lib/icons.ts`,
        "@/shared/lib": `${WINDOW}/shared/lib/index.ts`,
      }),
      moduleOf(`${WINDOW}/features/tree/model/tree.ts`, { "@/shared/lib/icons": `${WINDOW}/shared/lib/icons.ts` }),
      moduleOf(`${WINDOW}/pages/home/ui/Home.vue`, { "@/shared/lib/format": `${WINDOW}/shared/lib/format.ts` }),
      moduleOf(`${WINDOW}/pages/about/ui/About.vue`, { "@/shared/lib/format": `${WINDOW}/shared/lib/format.ts` }),
      moduleOf(`${WINDOW}/pages/home/model/home.ts`, { "@/shared/lib/paths": `${WINDOW}/shared/lib/paths.ts` }),
      moduleOf(`${WINDOW}/app/routes.ts`, { "@/shared/lib/paths": `${WINDOW}/shared/lib/paths.ts`, "@/shared/lib/boot": `${WINDOW}/shared/lib/boot.ts` }),
      moduleOf(`${WINDOW}/app/App.vue`, { "@/shared/lib/format": `${WINDOW}/shared/lib/format.ts` }),
      moduleOf(`${WINDOW}/app/app.test.ts`, { "@/shared/lib/icons": `${WINDOW}/shared/lib/icons.ts` }),
      moduleOf(`${WINDOW}/shared/lib/index.ts`, { "./ids": `${WINDOW}/shared/lib/ids.ts` }),
      moduleOf(`${WINDOW}/shared/lib/icons.ts`),
      moduleOf(`${WINDOW}/shared/lib/format.ts`),
      moduleOf(`${WINDOW}/shared/lib/paths.ts`),
      moduleOf(`${WINDOW}/shared/lib/boot.ts`),
      moduleOf(`${WINDOW}/shared/lib/ids.ts`),
    ];
    const { findings, read } = findSharedServingOneSlice(modules, WINDOW);
    assert.deepEqual(
      findings.map((one) => `${one.file} ${one.symbol}`),
      [
        `${WINDOW}/shared/lib/index.ts serves only features/tree; it belongs in that slice`,
        `${WINDOW}/shared/lib/icons.ts serves only features/tree; it belongs in that slice`,
        `${WINDOW}/shared/lib/paths.ts serves only pages/home; it belongs in that slice`,
        `${WINDOW}/shared/lib/boot.ts serves only app; it belongs in that layer`,
        `${WINDOW}/shared/lib/ids.ts serves only features/tree; it belongs in that slice`,
      ],
    );
    assert.equal(read, 6);
  });
});

describe("the app layer rule", () => {
  const lines = (count) => "const a = 1;\n".repeat(count);

  it("fails a file over the limit, a composable file and a composable exported from another file", () => {
    const { findings, read } = findAppOverreach([
      { file: `${WINDOW}/app/window/createBig.ts`, text: lines(APP_MAX_LINES + 1) },
      { file: `${WINDOW}/app/window/useTabs.ts`, text: lines(3) },
      { file: `${WINDOW}/app/window/createWindow.ts`, text: "const x = 1;\nexport function useSneaky() {}\n" },
      { file: `${WINDOW}/app/window/createCore.ts`, text: lines(APP_MAX_LINES) },
    ]);
    assert.deepEqual(
      findings.map((one) => `${one.file.split("/").at(-1)}:${one.line} ${one.symbol}`),
      [
        `createBig.ts:${APP_MAX_LINES + 1} ${APP_MAX_LINES + 1} lines, at most ${APP_MAX_LINES} are allowed in app`,
        "useTabs.ts:1 a composable file in app",
        "createWindow.ts:2 composable useSneaky is exported from app",
      ],
    );
    assert.equal(read, 4);
  });

  it("leaves tests, stories and the testing folder out of what it reads", () => {
    const { findings, read } = findAppOverreach([
      { file: `${WINDOW}/app/window/useTabs.test.ts`, text: lines(APP_MAX_LINES * 2) },
      { file: `${WINDOW}/app/screens.stories.ts`, text: lines(APP_MAX_LINES * 2) },
      { file: `${WINDOW}/app/App.vue`, text: lines(5) },
    ]);
    assert.deepEqual(findings, []);
    assert.equal(read, 1);
  });

  it("counts a file that ends without a newline by its last line", () => {
    const { findings } = findAppOverreach([{ file: `${WINDOW}/app/a.ts`, text: lines(3).trimEnd() }], 3);
    assert.deepEqual(findings, []);
  });
});

describe("the late binding rule of the app layer", () => {
  const file = `${WINDOW}/app/window/createWindow.ts`;
  const symbols = (text) => findLateBindings([{ file, text }]).findings.map((one) => `${one.line} ${one.symbol}`);

  it("fails a holder, its assignment and an Object.assign onto an object made before", () => {
    const text = ["interface Holder { current?: Host }", "hostHolder.current = pluginHost", "Object.assign(services, { vaults })"].join("\n");
    assert.deepEqual(symbols(text), ["1 a holder with an optional current", "2 a holder assigned after it was made", "3 Object.assign fills an object that was made before"]);
  });

  it("fails a provide call on a state, wherever it is written", () => {
    assert.deepEqual(symbols("base.plugins.pluginServices.provide({ vaults })"), ["1 provide fills a state that was made before"]);
    assert.deepEqual(symbols("const run = () => {\n  facts\n    .provide(next)\n}"), ["2 provide fills a state that was made before"]);
  });

  it("fails a ref, shallowRef or reactive that starts empty or as a stub of empty functions", () => {
    const text = [
      "const stored = shallowRef<Services>({})",
      "const source = shallowRef<Source>({ getCommands: () => [], getValues: () => ({}) })",
      "const state = reactive({})",
      "const list = ref({ read() { return undefined } })",
    ].join("\n");
    assert.deepEqual(symbols(text), [
      "1 shallowRef starts as an empty stub that is filled later",
      "2 shallowRef starts as an empty stub that is filled later",
      "3 reactive starts as an empty stub that is filled later",
      "4 ref starts as an empty stub that is filled later",
    ]);
  });

  it("fails an interface whose fields are all optional", () => {
    assert.deepEqual(symbols("export interface Services {\n  notes?: Notes\n  files?: Files\n}"), ["1 interface Services has only optional fields"]);
  });

  it("reads the script of a component", () => {
    const text = "<template><i /></template>\n<script setup lang=\"ts\">\nconst stored = shallowRef({})\n</script>";
    assert.deepEqual(findLateBindings([{ file: `${WINDOW}/app/App.vue`, text }]).findings.map((one) => one.symbol), ["shallowRef starts as an empty stub that is filled later"]);
  });

  it("passes a state with its own methods, a made ref, a fresh object, a comparison, a comment and an interface with a required field", () => {
    const text = [
      "const services = createServices()",
      "const merged = Object.assign({}, a, b)",
      "const open = ref(false)",
      "const list = shallowRef<readonly string[]>([])",
      "const view = shallowRef({ name: \"one\" })",
      "provide(KEY, value)",
      "if (holder.current === undefined) return",
      "// the old holder.current = host is gone",
      "interface Deps { read: () => string; write?: () => void }",
      "interface Options { limit?: number }",
    ].join("\n");
    assert.deepEqual(symbols(text), []);
  });

  it("leaves a test, a story and a file outside the shipped sources alone", () => {
    const text = "holder.current = host";
    const files = [`${WINDOW}/app/a.test.ts`, `${WINDOW}/app/a.stories.ts`].map((one) => ({ file: one, text }));
    const { findings, read } = findLateBindings(files);
    assert.deepEqual(findings, []);
    assert.equal(read, 0);
  });
});

describe("the page rule of the layers around pages", () => {
  const at = (file, text) => findPageImports([{ file: `${WINDOW}/${file}`, text }]).findings.map((one) => one.symbol);

  it("fails a feature, an entity and a shared file that import a page", () => {
    assert.deepEqual(at("features/tabs/model/a.ts", "import { x } from \"@/pages/settings\""), ["features imports @/pages/settings"]);
    assert.deepEqual(at("entities/note/lib/a.ts", "export { x } from \"@/pages/welcome\""), ["entities imports @/pages/welcome"]);
    assert.deepEqual(at("shared/lib/a.ts", "import type { T } from \"@/pages/settings/types\""), ["shared imports @/pages/settings/types"]);
  });

  it("fails app that takes a state a page owns, a type of it, or the whole module", () => {
    assert.deepEqual(at("app/window/a.ts", "import { useSettings } from \"@/pages/settings\""), ["app takes useSettings from @/pages/settings"]);
    assert.deepEqual(at("app/window/a.ts", "import type { useSettings } from \"@/pages/settings\""), ["app takes useSettings from @/pages/settings"]);
    assert.deepEqual(at("app/window/a.ts", "import \"@/pages/settings\""), ["app imports @/pages/settings as a whole"]);
  });

  it("passes app that composes a page, draws a screen, and a file that imports no page", () => {
    assert.deepEqual(at("app/window/a.ts", "import { createSettingsPage } from \"@/pages/settings\""), []);
    assert.deepEqual(at("app/App.vue", "<script setup lang=\"ts\">\nimport { WelcomeScreen } from \"@/pages/welcome\"\n</script>"), []);
    assert.deepEqual(at("app/window/a.ts", "import { WelcomeScreen } from \"@/pages/welcome\""), ["app takes WelcomeScreen from @/pages/welcome"]);
    assert.deepEqual(at("features/tabs/model/a.ts", "import { x } from \"@/entities/tab\""), []);
  });

  it("leaves tests and stories alone, and counts what it read", () => {
    const text = "import { useSettings } from \"@/pages/settings\"";
    const { findings, read } = findPageImports([`${WINDOW}/app/a.test.ts`, `${WINDOW}/app/window/b.ts`].map((file) => ({ file, text })));
    assert.equal(findings.length, 1);
    assert.equal(read, 1);
  });
});

describe("the deep import rule of the test support", () => {
  const at = (text) => findDeepSliceImports([{ file: `${WINDOW}/testing/a.ts`, text }]).findings.map((one) => one.symbol);

  it("fails an import and a re-export of a file inside a slice", () => {
    assert.deepEqual(at("export { a } from \"@/entities/settings/lib/settingAt\""), ["@/entities/settings/lib/settingAt is not the index of a slice"]);
    assert.deepEqual(at("import type { B } from \"@/pages/settings/types\""), ["@/pages/settings/types is not the index of a slice"]);
  });

  it("passes the index of a slice and a shared file", () => {
    assert.deepEqual(at("import { a } from \"@/entities/settings\"\nimport { b } from \"@/shared/api/errors\""), []);
  });
});

describe("the logic rule of the app layer", () => {
  const symbols = (text, file = `${WINDOW}/app/window/createWindow.ts`) =>
    findAppLogic([{ file, text }]).findings.map((one) => `${one.line} ${one.symbol}`);

  it("fails a watcher, a listener and a timer, however the call is written", () => {
    assert.deepEqual(
      symbols(
        [
          "watch(() => a, run);",
          "watchEffect(run);",
          "globalThis.addEventListener('pagehide', run);",
          "window.removeEventListener('keydown', run);",
          "setTimeout(run, 10);",
          "globalThis.clearTimeout(timer);",
          "setInterval(run, 10);",
        ].join("\n"),
      ),
      [
        "1 calls watch",
        "2 calls watchEffect",
        "3 calls addEventListener",
        "4 calls removeEventListener",
        "5 calls setTimeout",
        "6 calls clearTimeout",
        "7 calls setInterval",
      ],
    );
  });

  it("finds a call in a component script and in a nested function", () => {
    const text = "<script setup lang=\"ts\">\nconst start = () => () => watch(a, b)\n</script>";
    assert.deepEqual(symbols(text, `${WINDOW}/app/App.vue`), ["2 calls watch"]);
  });

  it("fails a switch", () => {
    assert.deepEqual(symbols("switch (kind) { default: break }"), ["1 a switch"]);
  });

  it("allows one decision in a function and fails the second", () => {
    assert.deepEqual(symbols("const a = (x) => (x ? 1 : 2);"), []);
    assert.deepEqual(symbols("const a = (x) => {\n  if (x) return 1;\n  return x ? 2 : 3;\n};"), [
      "3 more than 1 decision in a function",
    ]);
  });

  it("counts a loop and a catch as decisions", () => {
    assert.deepEqual(symbols("const a = (xs) => {\n  for (const x of xs) run(x);\n  try { run(1) } catch { run(2) }\n};"), [
      "3 more than 1 decision in a function",
    ]);
  });

  it("counts the decisions of each function apart", () => {
    assert.deepEqual(symbols("const a = (x) => (x ? 1 : 2);\nconst b = (x) => (x ? 1 : 2);\nconst c = (x) => (x ? 1 : 2);"), []);
  });

  it("fails a file whose functions take more than the file bound together", () => {
    const text = [1, 2, 3, 4].map((n) => `const f${n} = (x) => (x ? 1 : 2);`).join("\n");
    assert.deepEqual(symbols(text), ["1 4 decisions, at most 3 are allowed in a file of app"]);
  });

  it("leaves tests, stories and the testing folder out of what it reads", () => {
    const { findings, read } = findAppLogic(
      ["a.test.ts", "a.stories.ts"].map((name) => ({ file: `${WINDOW}/app/${name}`, text: "watch(a, b);" })),
    );
    assert.deepEqual(findings, []);
    assert.equal(read, 0);
  });
});
