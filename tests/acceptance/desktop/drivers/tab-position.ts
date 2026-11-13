/**
 * The position a tab is left at: a page of a PDF or a book, a time of a recording. The kind of a tab
 * is read from the tab itself, and a position of the wrong kind for it fails.
 */
import path from 'node:path'
import { expect, type Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { readBox } from '../../../e2e/desktop/tests/support/browser.js'
import { openVaultFile } from '../../../e2e/desktop/tests/support/workspace.js'

import type { Host } from './components/host.js'
import { TabPanel, TabPanels } from './components/TabPanel.js'
import { TabStrip } from './components/TabStrip.js'
import { assertPosition, parseTime, type TabKind } from './rules.js'

const detectKind = async (panel: TabPanel): Promise<TabKind | undefined> => {
  if ((await panel.pageField().count()) > 0) return 'PDF Reader'
  if ((await panel.book().count()) > 0) return 'EPUB Reader'
  if ((await panel.player().count()) > 0) return 'Audio Player'
  return undefined
}

const readKind = async (panel: TabPanel, file: string, position: string): Promise<TabKind> => {
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

const leavePdfAt = async (page: Page, panel: TabPanel, file: string, target: number): Promise<void> => {
  const field = panel.pageField()
  for (let next = Number(await field.inputValue()) + 1; next <= target; next += 1) {
    await panel.nextPageButton().click()
    await expect(field).toHaveValue(String(next))
  }
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .toContain(`"page":${target}`)
}

const moveToFirstPane = async (host: Host, panel: TabPanel, file: string): Promise<void> => {
  const { page } = host
  const strip = new TabStrip(host)
  const first = strip.firstPane()
  if ((await strip.tabOfFileIn(first, file).count()) > 0) return
  await expect(panel.bookCount()).toHaveText(/^\d+ of \d+$/)
  const [, total = ''] = /^\d+ of (\d+)$/.exec(await panel.bookCount().innerText()) ?? []
  const from = await readBox(strip.tabOfFile(file))
  const to = await readBox(first)
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 15 })
  await page.mouse.up()
  await expect(strip.tabOfFileIn(first, file)).toHaveCount(1)
  await expect(panel.bookCount()).toHaveText(new RegExp(`^\\d+ of (?!${total}$)\\d+$`))
}

const leaveBookAt = async (page: Page, panel: TabPanel, file: string, target: number): Promise<void> => {
  const count = panel.bookCount()
  const before = (await readSavedStates(page.request)).get(file)
  for (let next = 2; next <= target; next += 1) {
    await panel.book().press('PageDown')
    await expect(count).toHaveText(new RegExp(`^${next} of \\d+$`))
  }
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .not.toBe(before)
}

const leaveRecordingAt = async (page: Page, panel: TabPanel, file: string, time: string): Promise<void> => {
  const player = panel.player()
  const slider = player.getByRole('slider')
  // A seek before the recording has loaded its length is lost.
  await expect(slider).not.toHaveAttribute('max', '0')
  await slider.fill(String(parseTime(time) * 1000))
  await expect(player).toContainText(time)
  await expect
    .poll(async () => (await readSavedStates(page.request)).get(file))
    .toContain(`"t":[${parseTime(time)},`)
}

export const openTabAt = async (host: Host, file: string, position: string, where: string): Promise<void> => {
  const { page } = host
  const panel = new TabPanel(await openVaultFile(page, path.dirname(file), path.basename(file)))
  const kind = await readKind(panel, file, position)
  if (kind === 'PDF Reader') await leavePdfAt(page, panel, file, Number(where))
  else if (kind === 'EPUB Reader') {
    await moveToFirstPane(host, panel, file)
    await leaveBookAt(page, panel, file, Number(where))
  }
  else await leaveRecordingAt(page, panel, file, where)
}

/** Checks that the tab of the file is at the position. */
export const expectTabAt = async (host: Host, file: string, position: string, where: string): Promise<void> => {
  await new TabStrip(host).tabOfFile(file).click()
  const panel = new TabPanel(new TabPanels(host).ofFile(file))
  const kind = await readKind(panel, file, position)
  if (kind === 'PDF Reader') await expect(panel.pageField()).toHaveValue(where)
  else if (kind === 'EPUB Reader') await expect(panel.bookCount()).toHaveText(new RegExp(`^${where} of \\d+$`))
  else await expect(panel.player()).toContainText(where)
}
