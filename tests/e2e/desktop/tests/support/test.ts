/**
 * The `test` of ./certificate.ts, with the open tabs the core keeps for the vault cleared before
 * each test. The window opens the tabs the core kept, and every test shares the vault, so each one
 * starts from the tabs a new vault opens with.
 */
import fs from 'node:fs'
import https from 'node:https'
import { expect, getCertificateFile, test as base, type APIRequestContext } from './certificate.js'

export * from 'playwright/test'
export { getCertificateFile, getKeyFile } from './certificate.js'

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

/** The vault fixture whose text cannot be read, which the core reports as a failed task in every run. */
const UNREADABLE_FIXTURE = 'corrupted.pdf'

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

const readFailedTasks = async (baseURL: string): Promise<string[]> =>
  (await readTasks(baseURL))
    .filter((one) => one.error)
    .map((one) => [one.title, one.about, one.error].filter(Boolean).join(': '))
    .filter((text) => !text.includes(UNREADABLE_FIXTURE))

export const test = base.extend<{ isFreshTabs: void; hasNoFailedTask: void }>({
  hasNoFailedTask: [
    async ({ baseURL }, use) => {
      await use()
      await expect
        .poll(() => readFailedTasks(baseURL ?? ''), {
          message: 'the test leaves a failed task in the list of background tasks',
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
