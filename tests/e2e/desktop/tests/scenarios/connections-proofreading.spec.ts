import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, type Page } from '../support/test.js'
import { callRPC } from '../support/vault.js'
import {
  getPaletteField,
  openVaultFile,
  openWorkspace,
  runCommand,
  runCommandOverItem,
} from '../support/workspace.js'

const PROOFREADING = 'md.numen.tools.proofreading'
const SERVICE_TYPE = 'md.numen.connections.llm/openai-compatible'
const MOCK_TRANSCRIPT = /Mock transcribed text/
const PLUGIN_DIR = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../../../../plugins/connections-llm',
)

interface FakeServer {
  readonly address: string
  readonly process: ChildProcess
}

const buildFakeServer = (): string => {
  const prebuilt = path.join(PLUGIN_DIR, 'bin', 'fakeopenai')
  if (fs.existsSync(prebuilt)) return prebuilt
  const binary = test.info().outputPath('fakeopenai')
  fs.mkdirSync(path.dirname(binary), { recursive: true })
  execFileSync('go', ['build', '-o', binary, './backend/cmd/fakeopenai'], {
    cwd: PLUGIN_DIR,
    stdio: 'inherit',
  })
  return binary
}

const startFakeServer = (binary: string, args: readonly string[]): Promise<FakeServer> =>
  new Promise((resolve, reject) => {
    const child = spawn(binary, [...args], { stdio: ['ignore', 'pipe', 'inherit'] })
    child.once('error', reject)
    child.stdout?.once('data', (chunk: Buffer) =>
      resolve({ address: chunk.toString().trim(), process: child }),
    )
  })

const createConnection = async (page: Page, name: string, address: string): Promise<string> => {
  const answer = await callRPC(page.request, 'ProviderConnectionService/CreateProviderConnection', {
    typeId: SERVICE_TYPE,
    name,
    fields: { baseUrl: address },
  })
  return (answer.connection as { id: string }).id
}

const writeRows = async (page: Page, connectionIds: readonly string[]): Promise<void> => {
  const rows = connectionIds.map((connection) => ({ connection, model: 'fake-model' }))
  await callRPC(page.request, 'SettingsService/WriteSettings', {
    settings: [{ at: [PROOFREADING, 'connections'], value: JSON.stringify(rows) }],
  })
}

const deleteConnection = async (page: Page, id: string): Promise<void> => {
  await callRPC(page.request, 'ProviderConnectionService/DeleteProviderConnection', { id })
}

const proofreadTranscript = async (page: Page, isTranscribed: boolean): Promise<void> => {
  await openWorkspace(page)
  const tab = await openVaultFile(page, 'assets', 'talk.mp3')
  if (!isTranscribed) await runCommand(page, 'Transcribe', /^Transcribe/)
  await expect(tab.getByText(MOCK_TRANSCRIPT)).toBeVisible({ timeout: 30_000 })
  await runCommandOverItem(page, 'Proofread', /^Proofread/, /\(current\)$/)
}

const closePalette = async (page: Page): Promise<void> => {
  const field = getPaletteField(page)
  for (let press = 0; press < 2 && (await field.isVisible()); press += 1) {
    await page.keyboard.press('Escape')
  }
  await expect(field).toBeHidden()
}

test.describe('Proofreading through connections', () => {
  test.describe.configure({ timeout: 180_000 })

  let limited: FakeServer
  let working: FakeServer

  test.beforeAll(async () => {
    const binary = buildFakeServer()
    limited = await startFakeServer(binary, ['-status', '429', '-retry-after', '600'])
    working = await startFakeServer(binary, [])
  })

  test.afterAll(() => {
    limited.process.kill()
    working.process.kill()
  })

  test('names the connection and the reason, then ends the check on the second row when the first answers 429', async ({
    page,
  }) => {
    const first = await createConnection(page, 'Limited service', limited.address)
    const second = await createConnection(page, 'Working service', working.address)
    try {
      await writeRows(page, [first])
      await proofreadTranscript(page, false)

      await page.locator('.tasks-status__trigger').click()
      const failures = page.locator('.task-popover .task-item--failed')
      await expect(
        failures
          .filter({ hasText: 'Proofreading talk.mp3' })
          .filter({ hasText: 'Limited service: rate limited' }),
      ).toHaveCount(1, { timeout: 60_000 })
      const dismiss = failures.getByRole('button', { name: 'Dismiss' })
      while ((await dismiss.count()) > 0) await dismiss.first().click()
      await page.keyboard.press('Escape')

      await writeRows(page, [first, second])
      await proofreadTranscript(page, true)

      await expect(async () => {
        await closePalette(page)
        await runCommand(page, 'Delete text', /^Delete text/)
        await expect(page.getByRole('option')).toHaveCount(2, { timeout: 2_000 })
      }).toPass({ timeout: 90_000 })
      await closePalette(page)
    } finally {
      await deleteConnection(page, first)
      await deleteConnection(page, second)
    }
  })
})
