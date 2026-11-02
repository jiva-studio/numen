import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from 'playwright/test'
import { planShards } from './scripts/shards.mjs'

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const baseURL = `https://127.0.0.1:${port}`

// `scripts/run-offline.mjs` runs one process for each engine and shard, each against the server of
// its own port. A shard runs the spec files that `scripts/shards.mjs` deals to it; a run without
// NUMEN_E2E_SHARD runs every file against one server.
const readShardFiles = (): string[] | undefined => {
  const named = process.env.NUMEN_E2E_SHARD
  if (!named) return undefined
  const [index, count] = named.split('/').map(Number)
  const shard = planShards(count)[index]
  if (!shard) throw new Error(`NUMEN_E2E_SHARD=${named} names no shard`)
  return shard.map((file) => fileURLToPath(new URL(file, import.meta.url)))
}

export default defineConfig({
  testDir: './tests',
  testMatch: readShardFiles(),
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  maxFailures: process.env.CI ? 10 : undefined,
  expect: {
    timeout: 20_000,
  },
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    // The runner names a file for each process when NUMEN_E2E_JSON_DIR is set: the seconds of the
    // spec files for `npm run shards:measure`.
    ...(process.env.NUMEN_E2E_JSON
      ? [['json', { outputFile: process.env.NUMEN_E2E_JSON }] as const]
      : []),
  ],
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
    // The server prints this once the vault is read and every chunk has its vector (see
    // scripts/start-server.mjs). Without a url, a port that is taken fails the start: the debug
    // binary cannot listen on it and the script ends.
    wait: { stdout: /numen e2e: the vault is indexed/ },
    // The log of the server shows in the report, so a server that did not start or went away is
    // explained by what it printed.
    stdout: 'pipe',
    stderr: 'pipe',
    // The first pass over the vault, of many servers started together.
    timeout: 200_000,
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
      // WebKit does not report a request it was holding as cancelled.
      testIgnore: '**/*.chromium.spec.ts',
    },
  ],
})
