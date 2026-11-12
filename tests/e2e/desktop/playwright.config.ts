import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from 'playwright/test'
import { planShards, readPathsFromEnv } from './scripts/shards.mjs'

const port = process.env.NUMEN_TEST_PORT ?? '3456'
const baseURL = `https://127.0.0.1:${port}`

const readShardFiles = (): string[] | undefined => {
  const named = process.env.NUMEN_E2E_SHARD
  const paths = readPathsFromEnv()
  if (!named && !paths) return undefined
  const [index, count] = named ? named.split('/').map(Number) : [0, 1]
  const shard =
    Number.isInteger(index) && index >= 0 ? planShards(count, paths).at(index) : undefined
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
    // A CI shard writes a blob report to PLAYWRIGHT_BLOB_OUTPUT_DIR, named by
    // PLAYWRIGHT_BLOB_OUTPUT_NAME; a final job merges them into one HTML report.
    ...(process.env.PLAYWRIGHT_BLOB_OUTPUT_DIR ? [['blob'] as const] : []),
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
    // Without a signal Playwright kills the script at once, and the certificate, its key and the
    // vault of the run stay in the temporary folder. SIGTERM reaches the script and the server.
    gracefulShutdown: { signal: 'SIGTERM', timeout: 10_000 },
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
      // The specs of tests/requests call the server without a window: one engine is enough.
      testIgnore: ['**/*.chromium.spec.ts', '**/requests/**'],
    },
  ],
})
