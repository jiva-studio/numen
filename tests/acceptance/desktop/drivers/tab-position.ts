/**
 * The position a tab is left at: a page of a PDF or a book, a time of a recording. The kind of a tab
 * is read from the tab itself, and a position of the wrong kind for it fails.
 */
import path from 'node:path'
import { expect, type Locator, type Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { readBox } from '../../../e2e/desktop/tests/support/browser.js'
import { openVaultFile } from '../../../e2e/desktop/tests/support/workspace.js'

import { assertPosition, getEscaped, getStem, parseTime, type TabKind } from './rules.js'

const getPanel = (page: Page, file: string): Locator =>
  page.getByRole('tabpanel', { name: new RegExp(getEscaped(getStem(file)), 'i') })

const getPageField = (panel: Locator): Locator => panel.getByRole('spinbutton', { name: 'Page' })

const getCount = (panel: Locator): Locator => panel.locator('.book__count')

const getPlayer = (panel: Locator): Locator => panel.getByRole('group', { name: 'Recording' })

const detectKind = async (panel: Locator): Promise<TabKind | undefined> => {
  if ((await getPageField(panel).count()) > 0) return 'PDF Reader'
  if ((await panel.locator('.book-tab').count()) > 0) return 'EPUB Reader'
  if ((await getPlayer(panel).count()) > 0) return 'Audio Player'
  return undefined
}

const readKind = async (panel: Locator, file: string, position: string): Promise<TabKind> => {
  let kind: TabKind | undefined
  await expect
    .poll(async () => {
      kind = await detectKind(panel)
      return kind
    })
    .toBeDefined()
  if (!kind) throw new Error(`the tab "${file}" has no known kind`)
  assertPosition(kind, position, file)
  return kind
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

const moveToFirstPane = async (page: Page, file: string): Promise<void> => {
  const tabName = new RegExp('^' + getEscaped(getStem(file)), 'i')
  const panes = page.locator('section[data-workspace-pane]')
  const first = panes.first()
  if ((await first.getByRole('tab', { name: tabName }).count()) > 0) return
  const from = await readBox(page.getByRole('tab', { name: tabName }))
  const to = await readBox(first)
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 15 })
  await page.mouse.up()
  await expect(first.getByRole('tab', { name: tabName })).toHaveCount(1)
}

const leaveBookAt = async (page: Page, panel: Locator, file: string, target: number): Promise<void> => {
  const count = getCount(panel)
  const before = (await readSavedStates(page.request)).get(file)
  for (let next = 2; next <= target; next += 1) {
    await panel.locator('.book-tab').press('PageDown')
    await expect(count).toHaveText(new RegExp(`^${next} of \\d+$`))
  }
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .not.toBe(before)
}

const leaveRecordingAt = async (page: Page, panel: Locator, file: string, time: string): Promise<void> => {
  const player = getPlayer(panel)
  await player.getByRole('slider').fill(String(parseTime(time) * 1000))
  await expect(player).toContainText(time)
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .toContain(`"t":[${parseTime(time)},`)
}

/** Opens the tab of the file and leaves it at the position. */
export const openTabAt = async (page: Page, file: string, position: string, where: string): Promise<void> => {
  const panel = await openVaultFile(page, path.dirname(file), path.basename(file))
  const kind = await readKind(panel, file, position)
  if (kind === 'PDF Reader') await leavePdfAt(page, panel, file, Number(where))
  else if (kind === 'EPUB Reader') {
    await moveToFirstPane(page, file)
    await leaveBookAt(page, panel, file, Number(where))
  }
  else await leaveRecordingAt(page, panel, file, where)
}

/** Checks that the tab of the file is at the position. */
export const expectTabAt = async (page: Page, file: string, position: string, where: string): Promise<void> => {
  await page.getByRole('tab', { name: new RegExp(`^${getEscaped(getStem(file))}`, 'i') }).click()
  const panel = getPanel(page, file)
  const kind = await readKind(panel, file, position)
  if (kind === 'PDF Reader') await expect(getPageField(panel)).toHaveValue(where)
  else if (kind === 'EPUB Reader') await expect(getCount(panel)).toHaveText(new RegExp(`^${where} of \\d+$`))
  else await expect(getPlayer(panel)).toContainText(where)
}
