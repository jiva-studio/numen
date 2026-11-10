import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { APP_MAX_LINES, findAppOverreach, findComponentsBelowUi, findCoreClientOutsideApi, findLateBindings, findSharedServingOneSlice, getSegment, getSliceOf } from "./frontendGraph.mjs";

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

  it("fails a holder, its assignment and an Object.assign onto an object made before", () => {
    const text = [
      "const hostHolder: { current?: PluginHost } = {}",
      "hostHolder.current = pluginHost",
      "Object.assign(services, { vaults })",
    ].join("\n");
    const { findings, read } = findLateBindings([{ file, text }]);
    assert.deepEqual(
      findings.map((one) => `${one.line} ${one.symbol}`),
      ["1 a holder with an optional current", "2 a holder assigned after it was made", "3 Object.assign fills an object that was made before"],
    );
    assert.equal(read, 1);
  });

  it("passes a state with methods, a fresh object, a comparison and a comment", () => {
    const text = [
      "const services = createServices()",
      "services.provide({ vaults })",
      "const merged = Object.assign({}, a, b)",
      "if (holder.current === undefined) return",
      "// the old holder.current = host is gone",
      " * a { current?: Host } is not made",
    ].join("\n");
    assert.deepEqual(findLateBindings([{ file, text }]).findings, []);
  });

  it("leaves a test, a story and a file outside the shipped sources alone", () => {
    const text = "holder.current = host";
    const files = [`${WINDOW}/app/a.test.ts`, `${WINDOW}/app/a.stories.ts`].map((one) => ({ file: one, text }));
    const { findings, read } = findLateBindings(files);
    assert.deepEqual(findings, []);
    assert.equal(read, 0);
  });
});
