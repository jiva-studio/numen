import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'playwright/test'
import { defineBddConfig } from 'playwright-bdd'
import base from '../../e2e/desktop/playwright.config.js'

// The scenarios of `features/` become Playwright tests in `.features-gen/` (`npm run gen`), which
// this config runs against the server, the vault and the engines of the e2e suite.
const testDir = defineBddConfig({
  featuresRoot: '../../..',
  features: '../../../features/**/*.feature',
  steps: '{steps,support}/**/*.ts',
  outputDir: '.features-gen',
})

// The flashcards window is a server of its own, one port above the editor's. Its vault is named
// "Test" like the vault of the editor, in a folder of the run, and it serves the certificate of the
// editor's server.
const editorPort = process.env.NUMEN_TEST_PORT ?? '3456'
const flashcardsPort = String(Number(editorPort) + 1)
process.env.NUMEN_FLASHCARDS_TEST_PORT = flashcardsPort
process.env.NUMEN_FLASHCARDS_VAULT_FOLDER = '1'
process.env.NUMEN_FLASHCARDS_VAULT ??= path.join(
  os.tmpdir(),
  `numen-acceptance-flashcards-${flashcardsPort}`,
  'Test',
)
process.env.NUMEN_E2E_SHARED_CERTIFICATE_PORT = editorPort

const flashcardsServer = {
  command: `node "${fileURLToPath(new URL('../../e2e/desktop/scripts/start-flashcards-server.mjs', import.meta.url))}"`,
  url: `https://127.0.0.1:${flashcardsPort}`,
  ignoreHTTPSErrors: true,
  reuseExistingServer: false,
  timeout: 60_000,
  gracefulShutdown: { signal: 'SIGTERM' as const, timeout: 10_000 },
}

export default defineConfig({
  ...base,
  testDir,
  testMatch: undefined,
  webServer: [...[base.webServer ?? []].flat(), flashcardsServer],
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
})
