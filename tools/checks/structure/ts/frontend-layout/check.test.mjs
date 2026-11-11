import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findUnplacedFolders, findUnplacedInLayers } from "./check.mjs";

const window = { root: "apps/w/src", layers: ["shared", "entities", "features", "pages", "app"] };
const symbols = (found) => found.map((one) => `${one.file}: ${one.symbol}`);

describe("the layers of a window", () => {
  it("fails a top-level folder that is no layer", () => {
    const files = ["apps/w/src/main.ts", "apps/w/src/helpers/a.ts", "apps/w/src/shared/lib/a.ts"];
    assert.equal(findUnplacedInLayers(files, window).length, 1);
    assert.match(symbols(findUnplacedInLayers(files, window))[0], /helpers\/ belongs to no layer/);
  });

  it("fails a file or an unknown folder directly in shared, and passes the segments", () => {
    const files = ["apps/w/src/shared/paths.ts", "apps/w/src/shared/notices/messages.ts", "apps/w/src/shared/lib/a.ts", "apps/w/src/shared/api/b.ts", "apps/w/src/shared/ui/C.vue"];
    assert.deepEqual(symbols(findUnplacedInLayers(files, window)), [
      "apps/w/src/shared/paths.ts: a file directly in shared/ belongs to no segment",
      "apps/w/src/shared/notices: folder shared/notices/ is no segment (lib, api, model, ui)",
    ]);
  });

  it("fails a folder of a sliced layer that has no index.ts and no segment, and a loose file", () => {
    const files = ["apps/w/src/features/tab/index.ts", "apps/w/src/features/tab/model/a.ts", "apps/w/src/features/misc/a.ts", "apps/w/src/features/helper.ts", "apps/w/src/entities/note/ui/N.vue"];
    assert.deepEqual(symbols(findUnplacedInLayers(files, window)), [
      "apps/w/src/features/helper.ts: a file directly in features/ belongs to no slice",
      "apps/w/src/features/misc: folder features/misc/ is no slice: no index.ts and no segment",
    ]);
  });
});

describe("the folders of a root without layers", () => {
  it("passes segments, support folders and slices, and fails a folder that is none", () => {
    const files = ["p/frontend/ui/A.vue", "p/frontend/testing/f.ts", "p/frontend/deck/index.ts", "p/frontend/review/model/a.ts", "p/frontend/floor/a.ts", "p/frontend/index.ts"];
    assert.deepEqual(symbols(findUnplacedFolders(files, { root: "p/frontend" })), ["p/frontend/floor: folder floor/ is no segment and no slice: no index.ts and no segment of its own"]);
  });
});
