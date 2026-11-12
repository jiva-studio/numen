import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { serveRuntime } from '../../support/runtime.js'

const getButtons = (page: Page) => page.locator('[data-title-bar] [data-title-bar-command]')
const listButtonIds = (page: Page): Promise<string[]> =>
  getButtons(page).evaluateAll((all) =>
    all.map((one) => one.getAttribute('data-title-bar-command')!),
  )

test.describe('Plugin buttons in the title bar', () => {
  // The widths below count the three window buttons the bar draws on Windows.
  test.beforeEach(async ({ page }) => {
    await serveRuntime(page, 'windows')
  })

  test('moves the buttons back from the menu when the window grows', async ({ page }) => {
    await openWorkspace(page)
    const before = await listButtonIds(page)
    await page.setViewportSize({ width: 300, height: 800 })
    await expect(page.locator('[data-title-bar-more]')).toBeVisible()

    await page.setViewportSize({ width: 1280, height: 800 })

    await expect(page.locator('[data-title-bar-more]')).toHaveCount(0)
    await expect.poll(() => listButtonIds(page)).toStrictEqual(before)
  })

  test('reaches the buttons of the bar with the arrow keys', async ({ page }) => {
    await openWorkspace(page)
    const first = getButtons(page).first()
    await expect(page.locator('[data-title-bar] [role="toolbar"]')).toHaveAttribute(
      'aria-label',
      'Title bar',
    )

    const [, second] = await listButtonIds(page)
    await first.focus()
    await page.keyboard.press('ArrowRight')

    await expect
      .poll(() =>
        page.evaluate(() => document.activeElement?.getAttribute('data-title-bar-command')),
      )
      .toBe(second)
  })
})
