import { test, expect, type Locator, type Page } from '../../support/test.js'
import { readBox } from '../../support/browser.js'
import { getTab } from '../../support/layout.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'

/** The panes of the window, in the order they are drawn. */
const getPanes = (page: Page): Locator => page.locator('section[data-workspace-pane]')

/**
 * Whether the page is drawn under the middle of the reader showing, with an image that has
 * loaded and is not one flat colour. A page outside the visible part of the row is a black area
 * to the person.
 */
const isDrawnInView = (page: Page, at: number): Promise<boolean> =>
  page.locator('.reader__viewport:visible').evaluate(async (area, index) => {
    const frame = area.getBoundingClientRect()
    const centre = frame.left + frame.width / 2
    const sheet = area.querySelector<HTMLElement>(`.reader__page[data-page="${index}"]`)
    const picture = sheet?.querySelector<HTMLImageElement>('.reader__picture')
    if (!sheet || !picture?.complete || picture.naturalWidth === 0) return false
    const box = sheet.getBoundingClientRect()
    if (box.left > centre || box.right < centre) return false
    const canvas = document.createElement('canvas')
    canvas.width = 64
    canvas.height = 64
    const context = canvas.getContext('2d')!
    context.drawImage(picture, 0, 0, 64, 64)
    const data = context.getImageData(0, 0, 64, 64).data
    const colours = new Set<number>()
    for (let at = 0; at < data.length; at += 4) {
      colours.add((data[at]! << 16) | (data[at + 1]! << 8) | data[at + 2]!)
    }
    return colours.size > 2
  }, at)

test.describe('A PDF tab that is moved', () => {
  test('still draws the page it was on after it is dragged to the other pane', async ({ page }) => {
    await openWorkspace(page)
    await expect(getPanes(page)).toHaveCount(2)
    await openVaultFile(page, 'assets', 'pages.pdf')
    const field = page.getByLabel('Page', { exact: true })
    await field.fill('12')
    await field.press('Enter')
    await expect.poll(() => isDrawnInView(page, 11), { timeout: 20_000 }).toBe(true)

    // Panes are taken by position: a locator that filters on the tab would follow it.
    const holderAt = await getPanes(page).evaluateAll((panes) =>
      panes.findIndex((one) =>
        [...one.querySelectorAll('[role="tab"]')].some((tab) =>
          /pages\.pdf/.test(tab.textContent ?? ''),
        ),
      ),
    )
    expect(holderAt).toBeGreaterThanOrEqual(0)
    const holder = getPanes(page).nth(holderAt)
    const other = getPanes(page).nth(holderAt === 0 ? 1 : 0)
    const from = await readBox(getTab(page, 'pages.pdf'))
    const to = await readBox(other)
    await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
    await page.mouse.down()
    await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 15 })
    await page.mouse.up()

    await expect(other.getByRole('tab', { name: /pages\.pdf/ })).toHaveCount(1)
    await expect(holder.getByRole('tab', { name: /pages\.pdf/ })).toHaveCount(0)

    // The page is drawn without the page being turned.
    await expect(field).toHaveValue('12')
    await expect.poll(() => isDrawnInView(page, 11), { timeout: 20_000 }).toBe(true)
  })
})
