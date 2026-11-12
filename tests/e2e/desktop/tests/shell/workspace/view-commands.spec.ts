import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'

/**
 * The file tree opens in a tab of its own from the palette and from the keystroke its plugin
 * declares. The manifests and the handlers of every view are tested in the plugins.
 */
const FILES_TAB = /^Files/

const expectOpened = async (page: Page, open: () => Promise<void>): Promise<void> => {
  const tabs = page.getByRole('tab', { name: FILES_TAB })
  while ((await tabs.count()) > 0) {
    const count = await tabs.count()
    await tabs.first().click()
    await page.keyboard.press('Control+Shift+w')
    await expect(tabs).toHaveCount(count - 1)
  }

  await open()

  await expect(tabs).toHaveCount(1)
}

test.describe('the file tree in a tab', () => {
  test('opens from the palette by "Open file tree"', async ({ page }) => {
    await openWorkspace(page)

    await expectOpened(page, () => runPaletteCommand(page, 'file tree', 'Open file tree'))
  })

  test('opens by Control+Shift+e', async ({ page }) => {
    await openWorkspace(page)

    await expectOpened(page, () => page.keyboard.press('Control+Shift+e'))
  })
})
