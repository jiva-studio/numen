import { defineConfig } from 'playwright/test'
import { defineBddConfig } from 'playwright-bdd'
import base from '../../e2e/desktop/playwright.config.js'

// The scenarios of `features/` become Playwright tests in `.features-gen/` (`npm run gen`), which
// this config runs with the engines of the e2e suite.
const testDir = defineBddConfig({
  featuresRoot: '../../..',
  // The phone features (features/mobile) are proven by Go integration tests and the emulator script, not by this desktop harness.
  features: '../../../features/{editor,flashcards}/**/*.feature',
  steps: '{steps,support}/**/!(*.test).ts',
  outputDir: '.features-gen',
})

const isVideoOn = Boolean(process.env.PLAYWRIGHT_VIDEO)
const isLive = process.env.NUMEN_ACCEPTANCE_LIVE === '1'

// Every scenario starts a world of its own (drivers/world.ts), with a server and a vault of its own,
// so the scenarios run side by side and need none of the e2e suite's shared servers.
export default defineConfig({
  ...base,
  testDir,
  testMatch: undefined,
  globalSetup: './support/global-setup.ts',
  fullyParallel: true,
  workers: 2,
  ...(isLive && { timeout: 240_000 }),
  webServer: undefined,
  use: {
    ...base.use,
    ...(isVideoOn && { video: 'on', trace: 'retain-on-failure' }),
    ...(isLive && { trace: 'off' }),
  },
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ...(isVideoOn
      ? [
          [
            'blob',
            {
              outputDir: 'acceptance-blobs',
              fileName: `run-${process.platform}-${process.pid}.zip`,
            },
          ] as const,
        ]
      : []),
  ],
})
