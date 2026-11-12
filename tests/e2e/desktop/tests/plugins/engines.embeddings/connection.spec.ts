import { execFileSync, spawn, type ChildProcess } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { test, expect, type Locator, type Page } from '../../support/test.js'
import { callRPC, writeSetting } from '../../support/vault.js'
import { openTaskList, openWorkspace, runCommand } from '../../support/workspace.js'

const EMBEDDINGS_ID = 'md.numen.engines.embeddings'
const STATS_PATH = '/md.numen.tools.index.stats/GetStats'
const MODEL = 'fixture-embedding'
const WIDTH = 12

const here = path.dirname(fileURLToPath(import.meta.url))
const providerRoot = path.resolve(here, '../../../../../../plugins/connections-llm')

interface FakeServer {
  readonly address: string
  stop(): void
}

const startFakeServer = async (binary: string, flags: readonly string[]): Promise<FakeServer> => {
  const child: ChildProcess = spawn(binary, ['-addr', '127.0.0.1:0', ...flags], {
    stdio: ['ignore', 'pipe', 'inherit'],
  })
  const address = await new Promise<string>((resolve, reject) => {
    child.once('error', reject)
    child.once('exit', () => reject(new Error('the fake OpenAI server ended before it listened')))
    child.stdout?.once('data', (chunk: Buffer) => resolve(chunk.toString().trim()))
  })
  return {
    address,
    stop: () => {
      child.kill()
    },
  }
}

const openSettings = async (page: Page): Promise<Locator> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  return page.getByRole('tabpanel', { name: /Settings/ })
}

const addConnection = async (settings: Locator, page: Page, name: string, address: string) => {
  await settings.getByRole('button', { name: /^Connections$/ }).click()
  await settings.getByRole('button', { name: 'Add connection' }).click()
  const dialog = page.getByRole('dialog')
  await dialog.locator('[data-slot="select"]').first().click()
  await page.getByRole('menuitemradio', { name: /^OpenAI-compatible server/ }).click()
  await dialog.getByLabel('Name', { exact: true }).fill(name)
  await dialog.getByLabel('Address', { exact: true }).fill(address)
  await dialog.getByRole('button', { name: 'Save' }).click()
  await expect(dialog).toHaveCount(0)
  await settings.getByRole('button', { name: 'All plugins' }).click()
}

const chooseConnection = async (settings: Locator, page: Page, name: string) => {
  await settings.getByRole('button', { name: /Vector Embeddings/ }).click()
  await settings.getByTestId('setting-source').locator('.settings__choice').click()
  await page.getByRole('menuitemradio', { name: /^Connection/ }).click()
  await settings.getByRole('button', { name: 'Add connection' }).click()
  await settings.getByRole('button', { name: 'Connection', exact: true }).click()
  await page.getByRole('menuitemradio', { name: new RegExp(`^${name}`) }).click()
  await settings.getByRole('button', { name: 'Model', exact: true }).click()
  const search = page.getByRole('combobox', { name: 'Search models' })
  await search.fill(MODEL)
  await search.press('Enter')
}

const readCoverage = async (page: Page): Promise<{ chunks: number; embedded: number }> => {
  const state = await callRPC(page.request, 'VaultService/GetVaultState', {})
  const coverage = (state['coverage'] ?? {}) as { chunkCount?: string; embeddedCount?: string }
  return { chunks: Number(coverage.chunkCount ?? 0), embedded: Number(coverage.embeddedCount ?? 0) }
}

const readIndexedWidth = async (page: Page): Promise<number> => {
  const response = await page.request.post(STATS_PATH, { data: {} })
  return Number(((await response.json()) as { dimensions?: number }).dimensions ?? 0)
}

test.describe('Embeddings through a connection', () => {
  let binary = ''
  const servers: FakeServer[] = []

  test.beforeAll(() => {
    const prebuilt = path.join(providerRoot, 'bin', 'fakeopenai')
    if (fs.existsSync(prebuilt)) {
      binary = prebuilt
      return
    }
    binary = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-fakeopenai-')), 'fakeopenai')
    execFileSync('go', ['build', '-o', binary, './backend/cmd/fakeopenai'], { cwd: providerRoot })
  })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, [EMBEDDINGS_ID, 'source'], 'local')
    for (const server of servers.splice(0)) server.stop()
    const all = await callRPC(request, 'ProviderConnectionService/ListProviderConnections', {})
    for (const one of (all['connections'] ?? []) as { id: string }[]) {
      await callRPC(request, 'ProviderConnectionService/DeleteProviderConnection', { id: one.id })
    }
  })

  test.afterAll(() => {
    fs.rmSync(path.dirname(binary), { recursive: true, force: true })
  })

  test('indexes the vault with the vectors of a connection made in the interface', async ({
    page,
  }) => {
    test.setTimeout(180_000)
    const fake = await startFakeServer(binary, ['-dimensions', String(WIDTH)])
    servers.push(fake)
    await openWorkspace(page)
    const settings = await openSettings(page)
    await addConnection(settings, page, 'Fixture service', fake.address)

    await chooseConnection(settings, page, 'Fixture service')
    await expect(page.getByRole('alertdialog')).toContainText(
      /This indexes \d+ notes through Fixture service\. It may cost money\./,
    )
    await page.getByRole('alertdialog').getByRole('button', { name: 'Index' }).click()

    await expect
      .poll(
        async () => {
          const { chunks, embedded } = await readCoverage(page)
          return chunks > 0 && chunks === embedded ? await readIndexedWidth(page) : 0
        },
        { timeout: 120_000 },
      )
      .toBe(WIDTH)
  })

  test('names the connection and the reason when the service answers 429', async ({ page }) => {
    test.setTimeout(120_000)
    const limited = await startFakeServer(binary, ['-status', '429', '-retry-after', '1'])
    servers.push(limited)
    await openWorkspace(page)
    const settings = await openSettings(page)
    await addConnection(settings, page, 'Limited service', limited.address)

    await chooseConnection(settings, page, 'Limited service')
    await page.getByRole('alertdialog').getByRole('button', { name: 'Index' }).click()

    const tasks = await openTaskList(page)
    await expect(tasks.getByText(/Limited service: rate limited/).first()).toBeVisible({
      timeout: 60_000,
    })
  })
})
