import { defineConfig } from 'playwright/test'
import { defineBddConfig } from 'playwright-bdd'
import base from '../../e2e/desktop/playwright.config.js'

// The scenarios of `features/` become Playwright tests in `.features-gen/` (`npm run gen`), which
// this config runs with the engines of the e2e suite.
const testDir = defineBddConfig({
  featuresRoot: '../../..',
  features: '../../../features/**/*.feature',
  steps: '{steps,support}/**/*.ts',
  outputDir: '.features-gen',
})

// A run with video (`npm run acceptance:report`) marks the actions in the video and writes the
// scenario report of report/.
const isVideoOn = Boolean(process.env.PLAYWRIGHT_VIDEO)

// Every scenario starts a world of its own (drivers/world.ts), with a server and a vault of its own,
// so the scenarios run side by side and need none of the e2e suite's shared servers.
export default defineConfig({
  ...base,
  testDir,
  testMatch: undefined,
  fullyParallel: true,
  workers: '100%',
  webServer: undefined,
  // The text of each step stays on the page for a second before the step runs.
  ...(isVideoOn && { timeout: 120_000 }),
  use: {
    ...base.use,
    ...(isVideoOn && {
      video: { mode: 'on', show: { actions: { duration: 400, position: 'top-right' } } },
    }),
  },
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ...(isVideoOn
      ? [['./report/scenario-report.ts', { outputFolder: 'acceptance-report' }] as const]
      : []),
  ],
})
