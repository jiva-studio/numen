import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { test, expect, type Locator, type Page } from '../../support/test.js'
import { startOwnServer, type InstalledPlugin, type OwnServer } from '../../support/server.js'
import { runCommand } from '../../support/workspace.js'
import type { JsonObject } from '../../support/json.js'

const ONE = 'md.numen.test.probe-one'
const TWO = 'md.numen.test.probe-two'

const createProbe = (id: string, title: string, log: string): InstalledPlugin => ({
  id,
  files: {
    // The plugin folder lies under the e2e package, whose package.json makes every file ESM; the probe is CommonJS.
    'package.json': JSON.stringify({ type: 'commonjs' }),
    'manifest.json': JSON.stringify({
      id,
      name: title,
      version: '2026.11.1',
      protocol: 1,
      description: 'Records the secrets it receives',
      capabilities: [],
      targets: ['desktop.editor'],
      activationEvents: ['onVectorsReady', 'onFile:md'],
      permissions: [],
      backend: { entrypoint: 'bin/probe' },
      secrets: [
        {
          name: 'apiKey',
          title: `${title} key`,
          description: `Key read by ${title}`,
          group: 'openrouter',
        },
      ],
    }),
    'bin/probe': `#!/usr/bin/env node
const http = require('node:http')
const fs = require('node:fs')
const out = ${JSON.stringify(log)}
const port = Number(process.argv.find((a) => a.startsWith('--port=')).slice(7))
const varint = (b, i) => { let r = 0, s = 0; for (;;) { const c = b[i++]; r += (c & 127) * 2 ** s; if (c < 128) return [r, i]; s += 7 } }
const fields = (b) => {
  const list = []
  let i = 0
  while (i < b.length) {
    let tag; [tag, i] = varint(b, i)
    if ((tag & 7) === 2) { let n; [n, i] = varint(b, i); list.push([tag >> 3, b.subarray(i, i + n)]); i += n }
    else if ((tag & 7) === 0) { let v; [v, i] = varint(b, i); list.push([tag >> 3, v]) }
    else throw new Error('wire type ' + (tag & 7))
  }
  return list
}
const names = ['', 'id', 'name', 'group', 'value']
const parse = (body) => ({
  secrets: fields(body).filter(([n]) => n === 4).map(([, e]) => {
    const entry = { name: '', group: '', values: [] }
    for (const [n, v] of fields(e)) {
      if (n === 1) entry.name = v.toString()
      else if (n === 3) entry.group = v.toString()
      else if (n === 2) {
        const one = {}
        for (const [m, x] of fields(v)) one[names[m]] = x.toString()
        entry.values.push(one)
      }
    }
    return entry
  }),
})
const server = http.createServer((req, res) => {
  const chunks = []
  req.on('data', (c) => chunks.push(c))
  req.on('end', () => {
    res.setHeader('content-type', 'application/proto')
    if (req.url.endsWith('/Configure')) {
      fs.appendFileSync(out, JSON.stringify(parse(Buffer.concat(chunks))) + '\\n')
      res.end(Buffer.from([0x08, 0x01]))
    } else if (req.url.endsWith('/Ping')) {
      res.end(Buffer.from([0x0a, 0x02, 0x6f, 0x6b, 0x10, 0x01]))
    } else if (req.url.endsWith('/Shutdown')) {
      res.end(Buffer.from([0x08, 0x01]))
      setTimeout(() => process.exit(0), 10)
    } else {
      res.statusCode = 404
      res.end()
    }
  })
})
server.listen(port, '127.0.0.1', () => console.log('NUMEN_READY port=' + server.address().port))
`,
  },
})

interface Received {
  readonly id: string
  readonly name: string
  readonly group: string
  readonly value: string
}

const readReceivedEntries = (log: string): Received[] => {
  if (!fs.existsSync(log)) return []
  const lines = fs
    .readFileSync(log, 'utf8')
    .split('\n')
    .filter((line) => line !== '')
  const last = lines.at(-1)
  if (last === undefined) return []
  const call = JSON.parse(last) as { secrets: { name: string; values: Received[] }[] }
  return call.secrets.find((one) => one.name === 'apiKey')?.values ?? []
}

/** The address of the server of this file, which the shared base address of the project does not name. */
let serverUrl = ''

const reopen = async (page: Page): Promise<void> => {
  await page.goto(serverUrl)
  await expect(page.getByRole('tablist').first()).toBeVisible()
}

const callOn = async (
  page: Page,
  method: string,
  data: object,
): Promise<JsonObject> => {
  const answer = await page.request.post(`${serverUrl}/numen.v1.SettingsService/${method}`, {
    data,
    headers: { 'Content-Type': 'application/json' },
  })
  const text = await answer.text()
  if (!answer.ok()) throw new Error(`${method} answered ${answer.status()}: ${text}`)
  return text ? (JSON.parse(text) as JsonObject) : {}
}

const createSecret = async (
  page: Page,
  group: string,
  name: string,
  value: string,
): Promise<string> => (await callOn(page, 'CreateSecret', { group, name, value }))['id'] as string

const allow = async (page: Page, secretId: string, pluginId: string): Promise<void> => {
  await callOn(page, 'AllowPluginSecret', {
    secretId,
    pluginId,
    name: 'apiKey',
    group: 'openrouter',
  })
}

const openSecretsPage = async (page: Page): Promise<ReturnType<Page['getByRole']>> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /^Secrets$/ }).click()
  return settings
}

const rowOf = (scope: Locator, name: string): Locator =>
  scope.getByRole('group', { name, exact: true })

const choose = async (page: Page, select: Locator, names: RegExp[]): Promise<void> => {
  await select.click()
  for (const name of names) await page.getByRole('option', { name }).click()
  await page.keyboard.press('Escape')
}

const takeRowAction = async (page: Page, row: Locator, item: RegExp): Promise<void> => {
  await row.getByRole('button', { name: 'Secret actions' }).click()
  await page.getByRole('menuitem', { name: item }).click()
}

test.describe('Secrets shared between plugins', () => {
  let server: OwnServer
  let folder: string
  let logs: Record<string, string>

  test.beforeAll(async () => {
    folder = fs.mkdtempSync(path.join(os.tmpdir(), 'numen-secrets-'))
    logs = { [ONE]: path.join(folder, 'one.log'), [TWO]: path.join(folder, 'two.log') }
    server = await startOwnServer([
      createProbe(ONE, 'Probe one', logs[ONE] as string),
      createProbe(TWO, 'Probe two', logs[TWO] as string),
    ])
    serverUrl = server.url
  })

  test.afterAll(async () => {
    await server.stop()
    fs.rmSync(folder, { recursive: true, force: true })
  })

  test('gives a secret entered once to two plugins and keeps its value out of the page', async ({
    page,
  }) => {
    const value = 'invented-harbour-key-one'
    await reopen(page)
    const settings = await openSecretsPage(page)
    const section = settings.getByRole('region', { name: 'openrouter' })
    const field = section.getByLabel(/^Value of /)
    await field.fill(value)
    await field.press('Enter')

    await choose(page, section.locator('[data-slot="select"]'), [/Probe one/, /Probe two/])

    for (const id of [ONE, TWO]) {
      await expect
        .poll(() => readReceivedEntries(logs[id] as string).map((one) => one.value))
        .toEqual([value])
    }
    await expect(section.getByLabel(/^Value of /)).toHaveValue('')
    expect(await page.content()).not.toContain(value)
    const answer = await callOn(page, 'GetSettings', {})
    expect(JSON.stringify(answer)).not.toContain(value)
  })

  test('changes which secret a plugin reads first when a secret moves up', async ({ page }) => {
    const first = await createSecret(page, 'openrouter', 'Alpha', 'invented-key-alpha')
    const second = await createSecret(page, 'openrouter', 'Bravo', 'invented-key-bravo')
    await allow(page, first, ONE)
    await allow(page, second, ONE)
    // The server also holds the secrets of the previous tests, so only these two are compared.
    const readOrder = (): string[] =>
      readReceivedEntries(logs[ONE] as string)
        .map((one) => one.id)
        .filter((id) => id === first || id === second)
    await expect.poll(readOrder).toEqual([first, second])

    await reopen(page)
    const settings = await openSecretsPage(page)
    await takeRowAction(page, rowOf(settings, 'Bravo'), /^move up$/i)

    await expect.poll(readOrder).toEqual([second, first])
  })

  test('lets a plugin read a secret of a group the person made, chosen from the menu on its page', async ({
    page,
  }) => {
    const value = 'invented-key-trip'
    const trip = await createSecret(page, 'travel', 'Trip', value)
    await reopen(page)
    await runCommand(page, 'Open settings', /^Open settings$/)
    const settings = page.getByRole('tabpanel', { name: /Settings/ })
    await settings.getByRole('button', { name: /Probe two/ }).click()
    await choose(page, settings.locator('.settings__secret'), [/Trip/])

    await expect.poll(() => readReceivedEntries(logs[TWO] as string).map((one) => one.id)).toContain(trip)
    expect(await page.content()).not.toContain(value)
  })

  test('stops giving a secret to the plugins after it is deleted and the confirmation is accepted', async ({
    page,
  }) => {
    const doomed = await createSecret(page, 'openrouter', 'Doomed', 'invented-key-doomed')
    await allow(page, doomed, ONE)
    await expect
      .poll(() => readReceivedEntries(logs[ONE] as string).map((one) => one.id))
      .toContain(doomed)

    await reopen(page)
    const settings = await openSecretsPage(page)
    await takeRowAction(page, rowOf(settings, 'Doomed'), /^delete/i)
    const confirmation = page.getByRole('alertdialog')
    await expect(confirmation).toContainText('Probe one')
    await confirmation.getByRole('button', { name: /delete/i }).click()

    await expect
      .poll(() => readReceivedEntries(logs[ONE] as string).map((one) => one.id))
      .not.toContain(doomed)
  })
})
