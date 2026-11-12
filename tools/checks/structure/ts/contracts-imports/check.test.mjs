import assert from "node:assert/strict";
import { describe, it } from "node:test";
import { findContractImports, verifyContractsImportedOnlyByPluginsAndApps } from "./check.mjs";

const GO_CONTRACTS = "github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts";

/** A tree with the contracts packages, the libraries that import them and the plugins and applications that may. */
const PLANTED = {
  "libs/plugin/sdk-go/contracts/notes/notes.go": `package notes\n\nimport _ "${GO_CONTRACTS}/shared"\n\ntype Heading struct{ Text string }\n`,
  "libs/plugin/sdk-ts/src/contracts/notes.ts": "import type { Shared } from './shared'\n\nexport interface Heading { text: Shared }\n",
  "libs/core/kernel/usecase/heading/heading.go": `package heading\n\nimport (\n\t"context"\n\n\t"${GO_CONTRACTS}/notes"\n)\n`,
  "libs/core/kernel/usecase/heading/heading_test.go": `package heading\n\nimport "${GO_CONTRACTS}"\n`,
  "libs/plugin/sdk-go/host/headings.go": `package host\n\nimport "${GO_CONTRACTS}/notes"\n`,
  "libs/plugin/sdk-go/contractsx/other.go": `package contractsx\n\nimport "${GO_CONTRACTS}x/other"\n`,
  "libs/plugin/sdk-ts/src/index.ts": "export * from './types'\nexport * from './contracts/notes'\n",
  "libs/plugin/loader/src/hosting/headings.ts": "import type { Heading } from '@numen/plugin-sdk/contracts/notes'\n\nexport type Row = Heading\n",
  "libs/core/client/src/headings.ts": "export async function load() {\n  return import('@numen/plugin-sdk/contracts')\n}\nexport type H = import('@numen/plugin-sdk/contracts/notes').Heading\n",
  "libs/ui/components/src/Heading.vue": "<template><p /></template>\n\n<script setup lang=\"ts\">\nimport type { Heading } from '@numen/plugin-sdk/contracts/notes'\n\ndefineProps<{ heading: Heading }>()\n</script>\n",
  "libs/core/protocol/index.ts": "export * from '@numen/plugin-sdk/contractsx'\n",
  "plugins/notes/backend/service/headings.go": `package service\n\nimport "${GO_CONTRACTS}/notes"\n`,
  "plugins/views-outline/frontend/api/headings.ts": "import type { Heading } from '@numen/plugin-sdk/contracts/notes'\n\nexport type Row = Heading\n",
  "apps/desktop/editor/src/headings.ts": "import type { Heading } from '@numen/plugin-sdk/contracts/notes'\n\nexport type Row = Heading\n",
};

const findPlantedImports = (files) => findContractImports(Object.keys(files), (file) => files[file]);

describe("only plugins and applications import the contracts", () => {
  it("reports every import of a contracts package by a library, in Go, TypeScript and Vue", () => {
    const { findings, read } = findPlantedImports(PLANTED);
    assert.deepEqual(
      findings.map(({ file, line }) => `${file}:${line}`),
      [
        "libs/core/kernel/usecase/heading/heading.go:6",
        "libs/core/kernel/usecase/heading/heading_test.go:3",
        "libs/plugin/sdk-go/host/headings.go:3",
        "libs/plugin/sdk-ts/src/index.ts:2",
        "libs/plugin/loader/src/hosting/headings.ts:1",
        "libs/core/client/src/headings.ts:2",
        "libs/core/client/src/headings.ts:4",
        "libs/ui/components/src/Heading.vue:4",
      ],
    );
    assert.equal(read, 9);
  });

  it("passes a tree without contracts", () => {
    const { findings, read } = findPlantedImports({ "libs/core/kernel/usecase/heading/heading.go": 'package heading\n\nimport "context"\n', "plugins/notes/frontend/index.ts": "export const a = 1\n" });
    assert.deepEqual(findings, []);
    assert.equal(read, 1);
  });

  it("passes on the repository", () => {
    verifyContractsImportedOnlyByPluginsAndApps();
  });
});
