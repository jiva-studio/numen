import { test, expect, type Page } from 'playwright/test'
import { openWorkspace, runPaletteCommand } from '../../support/workspace'

/**
 * Every view a plugin draws in a tab of its own, the command that opens it from the palette, and
 * the keystroke the plugin declares for it.
 */
const VIEWS = [
  {
    command: 'Show files',
    query: 'files',
    tab: /^Files/,
    keys: 'Control+Shift+e',
  },
  { command: 'New agent', query: 'agent', tab: /^Agent/, keys: 'Control+Shift+a' },
  { command: 'New plex', query: 'plex', tab: /^(Thermodynamics|Plex)/, keys: 'Control+Shift+x' },
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

  test('opens with its content in the first frame it is drawn in', async ({ page }) => {
    await openWorkspace(page)
    await page.evaluate(() => {
      const labels: string[] = []
      ;(window as unknown as { loadingLabels: string[] }).loadingLabels = labels
      const look = () => {
        for (const one of document.querySelectorAll('.loading-indicator')) {
          if (one.childElementCount > 0) labels.push(one.textContent ?? '')
        }
        requestAnimationFrame(look)
      }
      requestAnimationFrame(look)
    })

    for (const keys of ['Control+Shift+x', 'Control+Shift+a', 'Control+Shift+x']) {
      const before = await page.getByRole('tab').count()
      await page.keyboard.press(keys)
      await expect(page.getByRole('tab')).toHaveCount(before + 1)
    }
    await page.waitForTimeout(500)

    const labels = await page.evaluate(
      () => (window as unknown as { loadingLabels: string[] }).loadingLabels,
    )
    expect(labels).toStrictEqual([])
  })
})
