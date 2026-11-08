import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

import { test, expect } from '../../support/test.js'
import { startOwnServer, type OwnServer } from '../../support/server.js'
import { getPaletteField, expectWorkspaceDrawn } from '../../support/workspace.js'

/** How long the vault takes to be embedded on a loaded machine. */
const INDEXED_WITHIN_MS = 45_000

/** A sentence of a note of the test vault, and a word that note is searched by. */
const NOTE_TEXT = 'A measure of uncertainty'
const QUERY = 'uncertainty'

const readTexts = (log: string): string[] => {
  if (!fs.existsSync(log)) return []
  return fs
    .readFileSync(log, 'utf8')
    .split('\n')
    .filter((line) => line !== '')
    .flatMap((line) => (JSON.parse(line) as { texts: string[] }).texts)
}

test.describe('The role of a text given to the embedding model', () => {
  let server: OwnServer
  let log: string

  test.beforeAll(async () => {
    log = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'numen-embed-log-')), 'embed.log')
    server = await startOwnServer([], { env: { NUMEN_MOCK_EMBED_LOG: log } })
  })

  test.afterAll(async () => {
    await server.stop()
    fs.rmSync(path.dirname(log), { recursive: true, force: true })
  })

  test('embeds a note as a passage and a typed query as a query', async ({ page }) => {
    await page.goto(server.url)
    await expectWorkspaceDrawn(page)

    // The index embeds the notes of the vault as documents.
    await expect
      .poll(() => readTexts(log).some((text) => text.includes(NOTE_TEXT)), {
        timeout: INDEXED_WITHIN_MS,
      })
      .toBe(true)
    const note = readTexts(log).filter((text) => text.includes(NOTE_TEXT))
    expect(note.every((text) => text.startsWith('passage: '))).toBe(true)

    // A query typed in the palette is embedded as a query.
    await page.keyboard.press('Control+k')
    const field = getPaletteField(page)
    await expect(field).toBeVisible()
    await field.fill(QUERY)
    await expect
      .poll(() => readTexts(log), { timeout: INDEXED_WITHIN_MS })
      .toContain(`query: ${QUERY}`)

    // Nothing reached the model without the prefix of its role.
    const texts = readTexts(log)
    expect(
      texts.filter((text) => !text.startsWith('passage: ') && !text.startsWith('query: ')),
    ).toEqual([])
    // A query is never embedded as a passage, and a note never as a query.
    expect(texts).not.toContain(`passage: ${QUERY}`)
    expect(texts.filter((text) => text.startsWith('query: ')).join('\n')).not.toContain(NOTE_TEXT)
  })
})
