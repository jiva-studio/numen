import { test, expect, type Locator } from '../../support/test.js'
import { createUniqueTitle, destroyIndexedNote, writeSetting } from '../../support/vault.js'
import { createNoteWithBody } from '../../support/layout.js'
import { openWorkspace } from '../../support/workspace.js'
import { waitForCurrentVectors } from '../../support/index-stats.js'

const EMBEDDINGS_ID = 'md.numen.engines.embeddings'
const MIN_CHUNK_LENGTH = [EMBEDDINGS_ID, 'minChunkLength']
const DEFAULT_MIN_CHUNK_LENGTH = 40

/** A body of 10 characters, below the default of 40. */
const SHORT_BODY = 'Tide table'

/** A body of 100 characters, above the default of 40. */
const LONG_BODY =
  'The ferry crosses the estuary at dawn with barley, salted fish and the post for the northern shores.'

const getItem = (page: Parameters<typeof openWorkspace>[0]): Locator =>
  page.locator('button:has(.index-stats__figure)')

const readVectors = async (item: Locator): Promise<number> => {
  const text = (await item.textContent()) ?? ''
  return Number(text.replace('⬡', '').replace(/\D/g, ''))
}

const isSettled = async (item: Locator): Promise<boolean> => {
  const text = (await item.textContent()) ?? ''
  return /\d/.test(text) && (await item.locator('.index-stats__trigger--indexing').count()) === 0
}

const readSettledVectors = async (item: Locator): Promise<number> => {
  await expect.poll(() => isSettled(item), { timeout: 45_000 }).toBe(true)
  return readVectors(item)
}

const readNoteCount = async (popover: Locator): Promise<number> => {
  const row = popover.locator('.index-popover__row--nested', { hasText: 'Notes' })
  const text = (await row.textContent()) ?? ''
  return Number(/·\s*([\d,.\s]+)\s*files/.exec(text)?.[1]?.replace(/\D/g, '') ?? 0)
}

test.describe('The smallest piece of text to index for meaning', () => {
  const notes: string[] = []

  test.afterEach(async ({ request }) => {
    await writeSetting(request, MIN_CHUNK_LENGTH, DEFAULT_MIN_CHUNK_LENGTH)
    for (const path of notes.splice(0)) await destroyIndexedNote(request, path)
  })

  test('gives a short piece no vector, until the setting lets it have one', async ({
    page,
    request,
  }) => {
    test.setTimeout(180_000)
    expect(SHORT_BODY).toHaveLength(10)
    expect(LONG_BODY).toHaveLength(100)
    await openWorkspace(page)
    const item = getItem(page)
    await waitForCurrentVectors(page)
    const before = await readSettledVectors(item)

    // With the default, the note of 100 characters gets a vector and the note of 10 does not.
    const longPath = await createNoteWithBody(
      request,
      createUniqueTitle('Long ferry note'),
      'notes',
      LONG_BODY,
    )
    notes.push(longPath)
    await item.click()
    const popover = page.getByRole('dialog', { name: 'Index stats' })
    await expect(popover).toBeVisible()
    await expect.poll(() => readVectors(item), { timeout: 45_000 }).toBeGreaterThan(before)
    const withLong = await readSettledVectors(item)
    const longChunks = withLong - before
    expect(longChunks).toBeGreaterThan(0)

    const notesBefore = await readNoteCount(popover)
    const shortPath = await createNoteWithBody(
      request,
      createUniqueTitle('Short tide note'),
      'notes',
      SHORT_BODY,
    )
    notes.push(shortPath)
    // The short note is read into the index, and the count keeps its value.
    await expect
      .poll(() => readNoteCount(popover), { timeout: 45_000 })
      .toBeGreaterThan(notesBefore)
    expect(await readSettledVectors(item)).toBe(withLong)

    // With 0 every piece gets a vector, the short one too.
    await writeSetting(request, MIN_CHUNK_LENGTH, 0)
    await expect.poll(() => readVectors(item), { timeout: 60_000 }).toBeGreaterThan(withLong)
    const withShort = await readSettledVectors(item)
    expect(withShort).toBeGreaterThan(withLong)

    // With 40 again the sweep takes the vectors of the pieces below 40 away.
    await writeSetting(request, MIN_CHUNK_LENGTH, DEFAULT_MIN_CHUNK_LENGTH)
    await expect.poll(() => readVectors(item), { timeout: 60_000 }).toBe(withLong)
    expect(await readSettledVectors(item)).toBe(withLong)
  })
})
