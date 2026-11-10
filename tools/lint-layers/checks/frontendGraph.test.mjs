import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findComponentsBelowUi, findCoreClientOutsideApi, findSharedServingOneSlice, getSegment, getSliceOf } from "./frontendGraph.mjs";

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
