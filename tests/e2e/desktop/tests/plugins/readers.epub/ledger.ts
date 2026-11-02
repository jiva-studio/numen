/**
 * Opening the invented two-chapter ledger book in the window, for the specs of the EPUB reader.
 */
import type { Page } from 'playwright/test'
import { expect } from '../../support/test.js'
import { waitForWindow } from '../../support/workspace.js'

/** The path of the book the specs of turns open. */
export const LEDGER = 'assets/ledger.epub'

export const openLedger = async (page: Page): Promise<void> => {
  await page.goto('/')
  await waitForWindow(page)
  await page.getByRole('tab', { name: /^Files/ }).click()
  const assetsRow = page.locator('[data-tree-row="assets"]')
  await expect(assetsRow).toBeVisible()
  await assetsRow.press('Enter')
  const epubRow = page.locator('[data-tree-row="assets/ledger.epub"]')
  await expect(epubRow).toBeVisible()
  await epubRow.press('Enter')

  await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.book__count')).toHaveText(/^1 of \d+$/, { timeout: 10_000 })
  await expect(page.locator('.book__paper:not(.book__paper--unset)')).toBeVisible()
}

/** The point a press is made at, as a fraction of the reading area across. */
export const getAreaPoint = async (page: Page, fraction: number) => {
  const box = await page.locator('.book__area').boundingBox()
  if (!box) throw new Error('no reading area')
  return { x: box.x + box.width * fraction, y: box.y + box.height / 2 }
}
