/**
 * What the tree, layout and settings scenarios read back from the core on top of `vault.ts`, and
 * the gestures they make in the window.
 */
import { expect, type APIRequestContext, type Locator, type Page } from 'playwright/test'
import { callRPC, getConnection, createNote, writeNote } from './vault.js'

/** Creates a note of that title in the folder holding the body given, and returns its path. */
export const createNoteWithBody = async (
  request: APIRequestContext,
  title: string,
  folder: string,
  body: string,
): Promise<string> => {
  const path = await createNote(request, title, folder)
  await writeNote(request, path, body)
  return path
}

/** Whether each of two notes writes a record about the other. */
export const getConnectionSides = async (
  request: APIRequestContext,
  first: string,
  second: string,
): Promise<{ first: boolean; second: boolean }> => {
  const connection = await getConnection(request, first, second)
  return { first: connection.firstSide !== undefined, second: connection.secondSide !== undefined }
}

/** One tab as the core keeps it for the vault in the window. */
export interface OpenTab {
  readonly id: string
  readonly kind: string
  readonly path: string
  readonly title: string
}

/** The tabs the core keeps for the vault in the window, and the id of the active one. */
export const readOpenTabs = async (
  request: APIRequestContext,
): Promise<{ tabs: OpenTab[]; activeId: string }> => {
  const answer = await callRPC(request, 'WorkspaceService/ReadOpenTabs', {})
  const tabs = ((answer['tabs'] ?? []) as Partial<OpenTab>[]).map((one) => ({
    id: one.id ?? '',
    kind: one.kind ?? '',
    path: one.path ?? '',
    title: one.title ?? '',
  }))
  return { tabs, activeId: String(answer['activeId'] ?? '') }
}

/**
 * The tab of that title in the window. A linked tab is announced with "Linked tab" before its
 * title, and every tab with its close button after it.
 */
export const getTab = (page: Page, title: string): Locator => {
  const escaped = title.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return page.getByRole('tab', { name: new RegExp(`^(Linked tab )?${escaped} Close ${escaped}$`) })
}

/** The row of a file or folder in the files tab, by its path. */
export const getRow = (page: Page, path: string): Locator =>
  page.locator(`[data-tree-row="${path}"]`)

/** Shows the files tab, and opens the folders on the way to `path` in it. */
export const showInFiles = async (page: Page, folders: readonly string[]): Promise<void> => {
  await getTab(page, 'Files').click()
  for (const folder of folders) {
    const row = getRow(page, folder)
    await expect(row).toBeVisible()
    if ((await row.getAttribute('aria-expanded')) !== 'true') await row.press('Enter')
    await expect(row).toHaveAttribute('aria-expanded', 'true')
  }
}

/**
 * Presses a row, carries it past the drag threshold and drops over the middle of another row,
 * the way a pointer drags a row onto a folder.
 */
export const dragRowOnto = async (page: Page, row: Locator, onto: Locator): Promise<void> => {
  const from = await row.boundingBox()
  const to = await onto.boundingBox()
  if (!from || !to) throw new Error('a row to drag is not drawn')
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2 - 12, { steps: 4 })
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 12 })
  await page.mouse.up()
}

/** The width of an element as laid out, in CSS pixels. */
const readWidth = async (element: Locator): Promise<number> =>
  element.evaluate((one) => one.getBoundingClientRect().width)

/** How many samples in turn, a tenth of a second apart, must agree for a width to be settled. */
const SETTLED_SAMPLES = 5

/**
 * Waits until the element has the same width in several samples in turn, and gives that width. A
 * reader fits its pages to the pane it is measured in, and the pane keeps changing until it has
 * been laid out, so a width read before then is not the one the document is drawn at. Two samples
 * can agree before the pane has reported its new size, so a run of them is asked for.
 */
export const readSettledWidth = async (element: Locator): Promise<number> => {
  const widths: number[] = []
  await expect
    .poll(
      async () => {
        widths.push(await readWidth(element))
        const run = widths.slice(-SETTLED_SAMPLES)
        return run.length === SETTLED_SAMPLES && run.every((one) => one === run[0])
      },
      { intervals: [100] },
    )
    .toBe(true)
  return widths.at(-1) ?? 0
}

/** The height of an element as laid out, in CSS pixels. */
export const readHeight = async (element: Locator): Promise<number> =>
  element.evaluate((one) => one.getBoundingClientRect().height)

/** The first line of text of an editor: a box whose height follows the size of the text. */
export const getFirstLine = (editor: Locator): Locator => editor.locator('.cm-line').first()
