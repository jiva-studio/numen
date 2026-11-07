import { test, expect, type Locator, type Page } from '../../support/test.js'
import {
  createUniqueTitle,
  destroyIndexedNote,
  destroyNote,
  writeNote,
} from '../../support/vault.js'
import { createNoteWithBody } from '../../support/layout.js'
import { readBox } from '../../support/browser.js'
import { openWorkspace } from '../../support/workspace.js'
import { waitForCurrentVectors } from '../../support/index-stats.js'

/** The title bar item that counts the vectors of the index. */
const getItem = (page: Page): Locator => page.locator('.index-stats__label')

/** The vectors the item shows. */
const readVectors = async (item: Locator): Promise<number> => {
  const text = (await item.textContent()) ?? ''
  return Number(text.replace('⬡', '').replace(/\D/g, ''))
}

/** Whether the index is embedded: the mark of the item does not pulse. */
const isSettled = async (item: Locator): Promise<boolean> => {
  const text = (await item.textContent()) ?? ''
  const isIndexing = await item.locator('.index-stats__icon--indexing').count()
  return /\d/.test(text) && isIndexing === 0
}

/** Waits until every chunk carries a vector, and returns the number of vectors then. */
const readSettledVectors = async (item: Locator): Promise<number> => {
  await expect.poll(() => isSettled(item), { timeout: 45_000 }).toBe(true)
  return readVectors(item)
}

/** Paragraphs that differ from each other, so each of them is a chunk of its own. */
const writeParagraphs = (topic: string, count: number, first = 1): string =>
  Array.from(
    { length: count },
    (_, i) =>
      `Paragraph ${first + i} on ${topic}: the ferry ${first + i} crosses the estuary at ${6 + i} in the morning, ` +
      `carrying ${100 * (first + i)} sacks of barley, ${first + i} crates of salted fish and the post for the ` +
      `villages along the northern shore, while the tide table of day ${first + i} is pinned to the mast.`,
  ).join('\n\n')

test.describe('the index stats item', () => {
  const notes: string[] = []

  test.afterEach(async ({ request }) => {
    for (const path of notes.splice(0)) await destroyIndexedNote(request, path)
  })

  test('shows the number of vectors in the title bar', async ({ page }) => {
    await openWorkspace(page)

    const item = getItem(page)
    await expect(item).toBeVisible()
    await expect(item).toHaveText(/^⬡\s+\d/)
    expect(await readSettledVectors(item)).toBeGreaterThan(0)
  })

  test('follows a note that is created, edited and removed', async ({ page, request }) => {
    await openWorkspace(page)
    const item = getItem(page)
    await waitForCurrentVectors(page)
    const before = await readSettledVectors(item)

    const title = createUniqueTitle('Estuary ferry log')
    const path = await createNoteWithBody(request, title, 'notes', writeParagraphs('barley', 6))
    notes.push(path)
    await expect.poll(() => readVectors(item), { timeout: 45_000 }).toBeGreaterThan(before)
    const created = await readSettledVectors(item)
    expect(created).toBeGreaterThan(before)

    await writeNote(request, path, writeParagraphs('barley', 14))
    await expect.poll(() => readVectors(item), { timeout: 45_000 }).toBeGreaterThan(created)
    const edited = await readSettledVectors(item)
    expect(edited).toBeGreaterThan(created)

    await destroyNote(request, path)
    notes.length = 0
    await expect.poll(() => readVectors(item), { timeout: 45_000 }).toBe(before)
  })

  test('opens the popover below the title bar, and closes it with Escape', async ({ page }) => {
    await openWorkspace(page)
    const item = getItem(page)
    await readSettledVectors(item)
    const trigger = page.getByRole('button', { name: / vectors/ })

    await trigger.click()

    await expect(trigger).toHaveAttribute('aria-expanded', 'true')
    const popover = page.getByRole('dialog', { name: 'Index stats' })
    await expect(popover).toBeVisible()
    await expect(popover.getByText('Vectors', { exact: true })).toBeVisible()
    await expect(popover.getByText('Chunks', { exact: true })).toBeVisible()

    const box = await readBox(popover)
    const bar = await readBox(page.locator('[data-title-bar]'))
    const viewport = page.viewportSize()
    expect(box.x).toBeGreaterThanOrEqual(0)
    expect(box.y).toBeGreaterThanOrEqual(0)
    expect(box.x + box.width).toBeLessThanOrEqual(viewport?.width ?? 0)
    expect(box.y).toBeGreaterThanOrEqual(bar.y + bar.height)
    expect(box.y + box.height).toBeLessThanOrEqual(viewport?.height ?? 0)
    const ground = await popover.evaluate((one) => getComputedStyle(one).backgroundColor)
    expect(ground).not.toBe('transparent')
    expect(ground).not.toMatch(/^rgba\(.*,\s*0\)$/)

    await page.keyboard.press('Escape')

    await expect(popover).toBeHidden()
    await expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })
})
