import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from 'playwright/test'

const port = process.env.NUMEN_FLASHCARDS_TEST_PORT ?? '3457'
const baseURL = `https://127.0.0.1:${port}`

// The vault the server copies the fixture into, named here so the tests read the same folder.
process.env.NUMEN_FLASHCARDS_VAULT ??= path.join(os.tmpdir(), `numen-e2e-flashcards-${port}`)

export default defineConfig({
  testDir: './flashcards',
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  expect: { timeout: 20_000 },
  reporter: [['list']],
  use: {
    baseURL,
    ignoreHTTPSErrors: true,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    viewport: { width: 760, height: 720 },
  },
  webServer: {
    command: `node "${fileURLToPath(new URL('./scripts/start-flashcards-server.mjs', import.meta.url))}"`,
    url: baseURL,
    ignoreHTTPSErrors: true,
    reuseExistingServer: false,
    timeout: 30_000,
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
})
