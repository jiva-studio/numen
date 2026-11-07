import { test, expect, type Locator } from '../../support/test.js'
import { createUniqueTitle, destroyIndexedNote, writeSetting } from '../../support/vault.js'
import { createNoteWithBody } from '../../support/layout.js'
import { openWorkspace } from '../../support/workspace.js'

const EMBEDDINGS_ID = 'md.numen.engines.embeddings'
const CHUNK_TOKENS = [EMBEDDINGS_ID, 'chunkTokens']
const DEFAULT_CHUNK_TOKENS = 256
/** A chunk of 64 tokens holds at most 128 characters, a quarter of the characters of the default. */
const LOW_CHUNK_TOKENS = 64

/** A note of about 2,600 characters in sentences of ordinary words, long enough to be cut at either size. */
const LONG_BODY = Array.from(
  { length: 40 },
  (_, i) =>
    `Barge ${i + 1} carries barley, salted fish and the post across the estuary before the tide turns.`,
).join(' ')

/** The title bar item that counts the vectors of the index. */
const getItem = (page: Parameters<typeof openWorkspace>[0]): Locator =>
  page.locator('.index-stats__label')

const readVectors = async (item: Locator): Promise<number> => {
  const text = (await item.textContent()) ?? ''
  return Number(text.replace('⬡', '').replace(/\D/g, ''))
}

/** Whether every chunk that can carry a vector carries one: the mark of the item does not pulse. */
const isSettled = async (item: Locator): Promise<boolean> => {
  const text = (await item.textContent()) ?? ''
  return /\d/.test(text) && (await item.locator('.index-stats__icon--indexing').count()) === 0
}

/** Waits until the index is embedded, and returns the number of vectors then. */
const readSettledVectors = async (item: Locator): Promise<number> => {
  await expect.poll(() => isSettled(item), { timeout: 45_000 }).toBe(true)
  return readVectors(item)
}

test.describe('The longest chunk of a note, in tokens', () => {
  const notes: string[] = []

  test.afterEach(async ({ request }) => {
    await writeSetting(request, CHUNK_TOKENS, DEFAULT_CHUNK_TOKENS)
    for (const path of notes.splice(0)) await destroyIndexedNote(request, path)
  })

  test('cuts the notes again when the setting is lowered, and back when it is restored', async ({
    page,
    request,
  }) => {
    test.setTimeout(240_000)
    await openWorkspace(page)
    const item = getItem(page)
    const before = await readSettledVectors(item)
    const path = await createNoteWithBody(
      request,
      createUniqueTitle('Barge log'),
      'notes',
      LONG_BODY,
    )
    notes.push(path)
    await expect.poll(() => readVectors(item), { timeout: 45_000 }).toBeGreaterThan(before)
    const initial = await readSettledVectors(item)

    // Shorter chunks give the note more of them, each with a vector.
    await writeSetting(request, CHUNK_TOKENS, LOW_CHUNK_TOKENS)
    await expect.poll(() => readVectors(item), { timeout: 90_000 }).toBeGreaterThan(initial)
    const lowered = await readSettledVectors(item)
    expect(lowered).toBeGreaterThan(initial)

    // The default again cuts the note as it was, and the vectors of the longer chunks are restored.
    await writeSetting(request, CHUNK_TOKENS, DEFAULT_CHUNK_TOKENS)
    await expect.poll(() => readVectors(item), { timeout: 90_000 }).toBe(initial)
    expect(await readSettledVectors(item)).toBe(initial)
  })
})
