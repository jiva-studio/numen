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

export default defineConfig({
  ...base,
  testDir,
  testMatch: undefined,
  // A scenario may wait for the engines to read a file and for the search to index it.
  timeout: 180_000,
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
})
