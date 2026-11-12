import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { test, expect, type Locator, type Page } from '../../support/test.js'
import { startOwnServer, type InstalledPlugin, type OwnServer } from '../../support/server.js'
import { runCommand } from '../../support/workspace.js'

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

const openSecretsPage = async (page: Page): Promise<ReturnType<Page['getByRole']>> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /^Secrets$/ }).click()
  return settings
}

const choose = async (page: Page, select: Locator, names: RegExp[]): Promise<void> => {
  await select.click()
  for (const name of names) await page.getByRole('option', { name }).click()
  await page.keyboard.press('Escape')
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

  test('gives a secret entered once on the page of secrets to two plugins', async ({
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
  })
})
