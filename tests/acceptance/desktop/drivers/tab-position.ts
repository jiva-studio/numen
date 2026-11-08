/**
 * The place a tab is left at: a page of a PDF or a book, a time of a recording. The kind of a tab
 * is read from the tab itself, and a place of the wrong kind for it fails.
 */
import path from 'node:path'
import { expect, type Locator, type Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { openVaultFile } from '../../../e2e/desktop/tests/support/workspace.js'

type TabKind = 'PDF Reader' | 'EPUB Reader' | 'Audio Player'

/** The kind of place each kind of tab has. */
const PLACES: Record<TabKind, string> = {
  'PDF Reader': 'page',
  'EPUB Reader': 'page',
  'Audio Player': 'time',
}

const getStem = (file: string): string => path.basename(file).replace(/\.[^.]+$/, '')

const getEscaped = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

const getPanel = (page: Page, file: string): Locator =>
  page.getByRole('tabpanel', { name: new RegExp(getEscaped(getStem(file)), 'i') })

const getPageField = (panel: Locator): Locator => panel.getByRole('spinbutton', { name: 'Page' })

const getReading = (panel: Locator): Locator => panel.locator('.book-tab__reading')

const getPlayer = (panel: Locator): Locator => panel.getByRole('group', { name: 'Recording' })

const detectKind = async (panel: Locator): Promise<TabKind | undefined> => {
  if ((await getPageField(panel).count()) > 0) return 'PDF Reader'
  if ((await panel.locator('.book-tab').count()) > 0) return 'EPUB Reader'
  if ((await getPlayer(panel).count()) > 0) return 'Audio Player'
  return undefined
}

/** The kind of the tab, once it is drawn; fails when the kind has no such place. */
const readKind = async (panel: Locator, file: string, place: string): Promise<TabKind> => {
  let kind: TabKind | undefined
  await expect
    .poll(async () => {
      kind = await detectKind(panel)
      return kind
    })
    .toBeDefined()
  if (!kind) throw new Error(`the tab "${file}" has no known kind`)
  if (PLACES[kind] !== place) {
    throw new Error(`the tab "${file}" is a ${kind} tab: its place is a ${PLACES[kind]}, not a ${place}`)
  }
  return kind
}

const getSeconds = (time: string): number => {
  const [minutes = 0, seconds = 0] = time.split(':').map(Number)
  return minutes * 60 + seconds
}

const leavePdfAt = async (page: Page, panel: Locator, file: string, target: number): Promise<void> => {
  const field = getPageField(panel)
  for (let next = Number(await field.inputValue()) + 1; next <= target; next += 1) {
    await panel.getByRole('button', { name: 'Next page' }).click()
    await expect(field).toHaveValue(String(next))
  }
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .toContain(`"page":${target}`)
}

const leaveBookAt = async (panel: Locator, target: number): Promise<void> => {
  const reading = getReading(panel)
  for (let next = 2; next <= target; next += 1) {
    await panel.locator('.book-tab').press('PageDown')
    await expect(reading).toContainText(new RegExp(`^${next} of \\d+`))
  }
}

const leaveRecordingAt = async (page: Page, panel: Locator, file: string, time: string): Promise<void> => {
  const player = getPlayer(panel)
  await player.getByRole('slider').fill(String(getSeconds(time) * 1000))
  await expect(player).toContainText(time)
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .toContain(`"t":[${getSeconds(time)},`)
}

/** Opens the tab of the file and leaves it at the place. */
export const openTabAt = async (page: Page, file: string, place: string, value: string): Promise<void> => {
  const panel = await openVaultFile(page, path.dirname(file), path.basename(file))
  const kind = await readKind(panel, file, place)
  if (kind === 'PDF Reader') await leavePdfAt(page, panel, file, Number(value))
  else if (kind === 'EPUB Reader') await leaveBookAt(panel, Number(value))
  else await leaveRecordingAt(page, panel, file, value)
}

/** Checks that the tab of the file is at the place. */
export const expectTabAt = async (page: Page, file: string, place: string, value: string): Promise<void> => {
  await page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(file))}`, 'i') }).click()
  const panel = getPanel(page, file)
  const kind = await readKind(panel, file, place)
  if (kind === 'PDF Reader') await expect(getPageField(panel)).toHaveValue(value)
  else if (kind === 'EPUB Reader') await expect(getReading(panel)).toContainText(new RegExp(`^${value} of \\d+`))
  else await expect(getPlayer(panel)).toContainText(value)
}
