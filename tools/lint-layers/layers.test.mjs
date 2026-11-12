import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import boundaries from "eslint-plugin-boundaries";
import { ESLint, Linter } from "eslint";
import tseslint from "typescript-eslint";
import { createDomainImports, createLayerConfig, createLayerElements, createLayerPolicies, createLibraryLayerConfigs } from "./layers.mjs";

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

describe("the layer hierarchy of a window, planted in files", () => {
  const order = ["shared", "entities", "features", "pages", "app"];
  const sliceLayers = ["entities", "features", "pages"];
  const files = {
    "src/shared/lib/clock.ts": "export const now = 1\n",
    "src/shared/lib/up.ts": "import { e } from '../../entities/ent'\nexport const up = e\n",
    "src/entities/ent/index.ts": "export { e } from './model/e'\n",
    "src/entities/ent/model/e.ts": "export const e = 1\n",
    "src/entities/ent/model/sideways.ts": "import { f } from '../../../features/fa/index'\nexport const s = f\n",
    "src/features/fa/index.ts": "export { f } from './model/f'\n",
    "src/features/fa/model/f.ts": "import { now } from '../../../shared/lib/clock'\nimport { e } from '../../../entities/ent/index'\nexport const f = now + e\n",
    "src/features/fb/index.ts": "export { g } from './model/g'\n",
    "src/features/fb/model/g.ts": "import { f } from '../../../features/fa/index'\nexport const g = f\n",
    "src/pages/pg/index.ts": "export { p } from './lib/p'\n",
    "src/pages/pg/lib/p.ts": "import { f } from '../../../features/fa/index'\nimport { f as deep } from '../../../features/fa/model/f'\nexport const p = f + deep\n",
  };
  const refused = {
    "src/shared/lib/up.ts": [1],
    "src/entities/ent/model/sideways.ts": [1],
    "src/features/fb/model/g.ts": [1],
    "src/pages/pg/lib/p.ts": [2],
  };

  it("refuses an upward import, a sibling slice and a deep import, and passes the allowed ones", async () => {
    const dir = fs.mkdtempSync(path.join(import.meta.dirname, ".fsd-self-test-"));
    try {
      for (const [name, text] of Object.entries(files)) {
        fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true });
        fs.writeFileSync(path.join(dir, name), text);
      }
      const layers = createLayerConfig({
        boundaries,
        elements: createLayerElements({ order, sliceLayers }),
        policies: createLayerPolicies({ order, sliceLayers, indexOnly: true }),
        categories: [{ category: "public", pattern: "src/*/*/index.ts" }],
      });
      layers.settings["boundaries/root-path"] = dir;
      const eslint = new ESLint({
        cwd: dir,
        overrideConfigFile: true,
        overrideConfig: [
          { files: ["**/*.ts"], languageOptions: { parser: tseslint.parser } },
          layers,
        ],
      });
      const results = await eslint.lintFiles(["src"]);
      const found = {};
      for (const result of results) {
        const lines = result.messages.filter((one) => one.ruleId?.startsWith("boundaries/")).map((one) => one.line);
        if (lines.length > 0) found[path.relative(dir, result.filePath)] = [...new Set(lines)];
      }
      assert.deepEqual(found, refused);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
