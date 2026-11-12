import { test, expect, type Locator, type Page } from '../../support/test.js'
import { readBox } from '../../support/browser.js'
import { getTab } from '../../support/layout.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'

const getPanes = (page: Page): Locator => page.locator('section[data-workspace-pane]')

const isDrawnInView = (page: Page, pageIndex: number): Promise<boolean> =>
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
    for (let offset = 0; offset < data.length; offset += 4) {
      colours.add((data[offset]! << 16) | (data[offset + 1]! << 8) | data[offset + 2]!)
    }
    return colours.size > 2
  }, pageIndex)

const dragToOtherPane = async (page: Page): Promise<void> => {
  // Panes are taken by position: a locator that filters on the tab would follow it.
  const holderIndex = await getPanes(page).evaluateAll((panes) =>
    panes.findIndex((one) =>
      [...one.querySelectorAll('[role="tab"]')].some((tab) =>
        /pages\.pdf/.test(tab.textContent ?? ''),
      ),
    ),
  )
  expect(holderIndex).toBeGreaterThanOrEqual(0)
  const panes = await getPanes(page).all()
  const sourcePane = panes[holderIndex]!
  const targetPane = panes[1 - holderIndex]!
  const from = await readBox(getTab(page, 'pages.pdf'))
  const to = await readBox(targetPane)
  await page.mouse.move(from.x + from.width / 2, from.y + from.height / 2)
  await page.mouse.down()
  await page.mouse.move(to.x + to.width / 2, to.y + to.height / 2, { steps: 15 })
  await page.mouse.up()

  await expect(targetPane.getByRole('tab', { name: /pages\.pdf/ })).toHaveCount(1)
  await expect(sourcePane.getByRole('tab', { name: /pages\.pdf/ })).toHaveCount(0)
}

test.describe('A PDF tab that is moved', () => {
  test('still draws the page it was turning to after it is dragged to the other pane in the middle of the turn', async ({
    page,
  }) => {
    await openWorkspace(page)
    await expect(getPanes(page)).toHaveCount(2)
    await openVaultFile(page, 'assets', 'pages.pdf')
    const field = page.getByLabel('Page', { exact: true })
    await field.fill('12')
    await field.press('Enter')

    // The row travels to the page for about a second; the tab is dragged while it is on the way.
    await dragToOtherPane(page)

    await expect(field).toHaveValue('12')
    await expect.poll(() => isDrawnInView(page, 11), { timeout: 20_000 }).toBe(true)
  })
})
