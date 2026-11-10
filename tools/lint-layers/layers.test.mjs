import assert from "node:assert/strict";
import { describe, it } from "node:test";
import boundaries from "eslint-plugin-boundaries";
import { Linter } from "eslint";
import { createDomainImports, createLibraryLayerConfigs } from "./layers.mjs";

const domainImports = createDomainImports({ message: "no domain", direction: "no application" });

const configs = createLibraryLayerConfigs({
  boundaries,
  order: ["shared", "features", "screens"],
  sliceLayers: ["features"],
  outsideLayers: ["src/**/*.stories.ts", "src/**/fixtures/**", "src/vite.ts"],
  domainImports,
});

/** The imports of the source that the library configuration rejects when it lives at `filename`. */
function findRejectedImports(filename, source) {
  const messages = new Linter().verify(source, configs, { filename });
  assert.deepEqual(
    messages.filter((message) => message.fatal),
    [],
  );
  return messages.filter((message) => message.ruleId === "no-restricted-imports").map((message) => message.line);
}

const DOMAIN_SOURCE = [
  "import { createClient } from '@numen/core-client'",
  "import { Note } from '@numen/protocol'",
  "import { sample } from '../../../../../../apps/desktop/editor/src/sample'",
  "import { Button } from '@numen/ui'",
  "import Local from './Local.vue'",
].join("\n");

describe("the import block of the stories and fixtures of a component library", () => {
  it("fails a wire type, a client and an application in a story", () => {
    assert.deepEqual(findRejectedImports("src/shared/ui/button/Button.stories.ts", DOMAIN_SOURCE), [1, 2, 3]);
  });

  it("fails them in a fixture, at any depth", () => {
    assert.deepEqual(findRejectedImports("src/shared/fixtures/rows.ts", DOMAIN_SOURCE), [1, 2, 3]);
    assert.deepEqual(findRejectedImports("src/features/tree/fixtures/deep/rows.ts", DOMAIN_SOURCE), [1, 2, 3]);
  });

  it("passes the imports a story may make", () => {
    const source = ["import { expect } from 'storybook/test'", "import Local from './Local.vue'", "import { DARK } from '@/shared/fixtures/theme'"].join("\n");
    assert.deepEqual(findRejectedImports("src/shared/ui/button/Button.stories.ts", source), []);
  });
});
