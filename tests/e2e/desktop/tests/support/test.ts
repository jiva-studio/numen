/**
 * Playwright's `test`, with the open tabs the core keeps for the vault cleared before each test.
 * The window opens the tabs the core kept, and every test shares the vault, so each one starts
 * from the tabs a new vault opens with.
 */
import { createHash, X509Certificate } from 'node:crypto'
import fs from 'node:fs'
import https from 'node:https'
import os from 'node:os'
import path from 'node:path'
import { expect, test as base, type APIRequestContext } from 'playwright/test'

export * from 'playwright/test'

/** Clears the open tabs the core keeps for the vault in the window. */
export const forgetOpenTabs = async (request: APIRequestContext): Promise<void> => {
  const answer = await request.post('/numen.v1.WorkspaceService/WriteOpenTabs', {
    data: { tabs: [], activeId: '' },
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`clearing the open tabs answered ${answer.status()}`)
}

/** The saved state of each open tab the core keeps for the vault in the window, by the tab's path. */
export const readSavedStates = async (request: APIRequestContext): Promise<Map<string, string>> => {
  const answer = await request.post('/numen.v1.WorkspaceService/ReadOpenTabs', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`reading the open tabs answered ${answer.status()}`)
  const { tabs = [] } = (await answer.json()) as { tabs?: { path?: string; state?: string }[] }
  return new Map(tabs.map((one) => [one.path ?? '', one.state ?? '']))
}

/** The path of the tab the core keeps as active for the vault in the window, empty for none. */
export const readActivePath = async (request: APIRequestContext): Promise<string> => {
  const answer = await request.post('/numen.v1.WorkspaceService/ReadOpenTabs', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  })
  if (!answer.ok()) throw new Error(`reading the open tabs answered ${answer.status()}`)
  const { tabs = [], activeId = '' } = (await answer.json()) as {
    tabs?: { id?: string; path?: string }[]
    activeId?: string
  }
  return tabs.find((one) => one.id === activeId)?.path ?? ''
}

/** The TLS certificate the debug server listening on port wrote, which scripts/start-server.mjs names. */
const getCertificateFile = (port: string): string =>
  path.join(os.tmpdir(), `numen-e2e-certificate-${port}.pem`)

/** The base64 SHA-256 of the public key of the certificate in file, the form of Chromium's SPKI list. */
const getPublicKeyHash = (file: string): string => {
  if (!fs.existsSync(file)) throw new Error(`the debug server wrote no certificate to ${file}`)
  const publicKey = new X509Certificate(fs.readFileSync(file)).publicKey
  return createHash('sha256')
    .update(publicKey.export({ type: 'spki', format: 'der' }))
    .digest('base64')
}

/** The vault fixture whose text cannot be read, which the core reports as a failed task in every run. */
const UNREADABLE_FIXTURE = 'corrupted.pdf'

/** The first list of tasks the editor window of the debug server reports, as a Connect stream. */
const readTasks = (
  baseURL: string,
): Promise<{ title?: string; about?: string; error?: string }[]> =>
  new Promise((resolve, reject) => {
    const origin = new URL(baseURL)
    const json = Buffer.from(JSON.stringify({ window: 'editor' }))
    const header = Buffer.alloc(5)
    header.writeUInt32BE(json.length, 1)
    const call = https.request(
      {
        host: origin.hostname,
        port: Number(origin.port),
        path: '/numen.v1.WindowService/WatchTasks',
        method: 'POST',
        ca: fs.readFileSync(getCertificateFile(origin.port)),
        headers: { 'Content-Type': 'application/connect+json' },
      },
      (answer) => {
        let received = Buffer.alloc(0)
        answer.on('data', (chunk: Buffer) => {
          received = Buffer.concat([received, chunk])
          if (received.length < 5 || received.length < 5 + received.readUInt32BE(1)) return
          call.destroy()
          const message = JSON.parse(received.subarray(5, 5 + received.readUInt32BE(1)).toString())
          resolve(
            (message as { tasks?: { title?: string; about?: string; error?: string }[] }).tasks ??
              [],
          )
        })
        answer.on('error', reject)
      },
    )
    call.on('error', reject)
    call.end(Buffer.concat([header, json]))
  })

/** The tasks that ended in failure, each as its title, what it is about and why it failed. */
export const readFailedTasks = async (baseURL: string): Promise<string[]> =>
  (await readTasks(baseURL))
    .filter((one) => one.error)
    .map((one) => [one.title, one.about, one.error].filter(Boolean).join(': '))
    .filter((text) => !text.includes(UNREADABLE_FIXTURE))

export const test = base.extend<{ isFreshTabs: void; hasNoFailedTask: void }>({
  // Chromium aborts the first handshake of each context with a certificate it does not trust, even
  // with ignoreHTTPSErrors. The server writes its certificate after the config is loaded, so the
  // worker reads it.
  launchOptions: [
    async ({ launchOptions, browserName }, use, workerInfo) => {
      if (browserName !== 'chromium') return use(launchOptions)
      const port = new URL(workerInfo.project.use.baseURL ?? '').port
      const publicKeyHash = getPublicKeyHash(getCertificateFile(port))
      await use({
        ...launchOptions,
        args: [
          ...(launchOptions.args ?? []),
          `--ignore-certificate-errors-spki-list=${publicKeyHash}`,
        ],
      })
    },
    { scope: 'worker' },
  ],
  hasNoFailedTask: [
    async ({ baseURL }, use) => {
      await use()
      await expect
        .poll(() => readFailedTasks(baseURL ?? ''), {
          message: 'the test leaves a failed task in the status bar',
        })
        .toEqual([])
    },
    { auto: true },
  ],
  isFreshTabs: [
    async ({ request }, use) => {
      await forgetOpenTabs(request)
      await use()
    },
    { auto: true },
  ],
})
