import { test, expect, type Page } from '../../support/test.js'
import { createUniqueTitle, destroyIndexedNote, writeSetting } from '../../support/vault.js'
import { createNoteWithBody } from '../../support/layout.js'
import { openWorkspace } from '../../support/workspace.js'
import { readNoteCounts, waitForCurrentVectors } from '../../support/index-stats.js'

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

const readNoteChunks = async (page: Page): Promise<number> => (await readNoteCounts(page)).chunks

/** Waits until no chunk of the index waits for a vector, and returns the chunks of the notes then. */
const readEmbeddedNoteChunks = async (page: Page): Promise<number> => {
  let chunks = 0
  await expect
    .poll(
      async () => {
        const counts = await readNoteCounts(page)
        chunks = counts.chunks
        return counts.isEmbedded
      },
      { timeout: 45_000 },
    )
    .toBe(true)
  return chunks
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
    await waitForCurrentVectors(page)
    const before = await readEmbeddedNoteChunks(page)
    const path = await createNoteWithBody(
      request,
      createUniqueTitle('Barge log'),
      'notes',
      LONG_BODY,
    )
    notes.push(path)
    await expect.poll(() => readNoteChunks(page), { timeout: 45_000 }).toBeGreaterThan(before)
    const initial = await readEmbeddedNoteChunks(page)

    // Shorter chunks give the note more of them, each with a vector.
    await writeSetting(request, CHUNK_TOKENS, LOW_CHUNK_TOKENS)
    await expect.poll(() => readNoteChunks(page), { timeout: 90_000 }).toBeGreaterThan(initial)
    const lowered = await readEmbeddedNoteChunks(page)
    expect(lowered).toBeGreaterThan(initial)

    // The default again cuts the note as it was, and the vectors of the longer chunks are restored.
    await writeSetting(request, CHUNK_TOKENS, DEFAULT_CHUNK_TOKENS)
    await expect.poll(() => readNoteChunks(page), { timeout: 90_000 }).toBe(initial)
    expect(await readEmbeddedNoteChunks(page)).toBe(initial)
  })
})
