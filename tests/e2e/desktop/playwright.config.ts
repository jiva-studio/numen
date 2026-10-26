import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from 'playwright/test'

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const baseURL = `https://127.0.0.1:${port}`

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  maxFailures: process.env.CI ? 10 : undefined,
  expect: {
    timeout: 20_000,
  },
  reporter: [['list'], ['html', { open: 'never', outputFolder: 'playwright-report' }]],
  use: {
    baseURL,
    ignoreHTTPSErrors: true,
    trace: process.env.PLAYWRIGHT_VIDEO ? 'on' : 'retain-on-failure',
    video: process.env.PLAYWRIGHT_VIDEO ? 'on' : 'off',
    screenshot: process.env.PLAYWRIGHT_VIDEO ? 'on' : 'only-on-failure',
    viewport: { width: 1280, height: 860 },
  },
  webServer: {
    command: `node "${fileURLToPath(new URL('./scripts/start-server.mjs', import.meta.url))}"`,
    url: baseURL,
    ignoreHTTPSErrors: true,
    reuseExistingServer: !process.env.CI,
    timeout: 30_000,
  },
  // The window is WebKitGTK on Linux and WKWebView on macOS. `scripts/run-offline.mjs` runs each
  // project against a server of its own, since a run changes the vault.
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'], deviceScaleFactor: 1 },
    },
  ],
})
