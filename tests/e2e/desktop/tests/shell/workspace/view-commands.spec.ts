import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'

/**
 * Every view a plugin draws in a tab of its own, the command that opens it from the palette, and
 * the keystroke the plugin declares for it.
 */
const VIEWS = [
  {
    command: 'Open file tree',
    query: 'file tree',
    tab: /^Files/,
    keys: 'Control+Shift+e',
  },
  { command: 'Open agent chat', query: 'agent', tab: /^Agent/, keys: 'Control+Shift+a' },
  {
    command: 'Open plex graph',
    query: 'plex',
    tab: /^(Thermodynamics|Plex)/,
    keys: 'Control+Shift+x',
  },
  { command: 'Record audio', query: 'record', tab: /^Record Audio/, keys: undefined },
] as const

/**
 * Closes every tab of the view, runs the opening, and checks it drew one.
 */
const expectOpened = async (page: Page, name: RegExp, open: () => Promise<void>): Promise<void> => {
  const tabs = page.getByRole('tab', { name })
  while ((await tabs.count()) > 0) {
    const count = await tabs.count()
    await tabs.first().click()
    await page.keyboard.press('Control+Shift+w')
    await expect(tabs).toHaveCount(count - 1)
  }

  await open()

  await expect(tabs).toHaveCount(1)
}

test.describe('a view a plugin draws in a tab', () => {
  for (const view of VIEWS) {
    test(`opens from the palette by "${view.command}"`, async ({ page }) => {
      await openWorkspace(page)

      await expectOpened(page, view.tab, () => runPaletteCommand(page, view.query, view.command))
    })

    if (view.keys) {
      test(`opens by ${view.keys}`, async ({ page }) => {
        await openWorkspace(page)

        await expectOpened(page, view.tab, () => page.keyboard.press(view.keys))
      })
    }
  }
})
