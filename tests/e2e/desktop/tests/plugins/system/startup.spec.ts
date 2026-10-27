import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace'

/** The words a plugin that did not start is reported in, as a notice or as a task. */
const PLUGIN_FAILURE = /stopped:|Plugin stopped|failed to start|failed to load/i

/** The stock plugins, started with the window and asked for what they serve. */
test.describe('the stock plugins after the window starts', () => {
  test('start without a failure, a markdown note open', async ({ page }) => {
    const failedFetches: string[] = []
    page.on('response', (response) => {
      if (response.url().includes('/plugin/') && response.status() >= 400) {
        failedFetches.push(`${response.status()} ${response.url()}`)
      }
    })
    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.locator('[data-tree-row="notes"]').press('Enter')
    await page.locator('[data-tree-row="notes/Entropy.md"]').press('Enter')
    await expect(page.locator('.note').first()).toBeVisible()
    // A failed launch for the note is reported within a second.
    await page.waitForTimeout(1_000)

    await expect(page.locator('.notice').filter({ hasText: PLUGIN_FAILURE })).toHaveCount(0)
    await page.locator('.status-bar__trigger').click()
    await expect(page.locator('.task-popover')).toBeVisible()
    // The vault holds a corrupted PDF on purpose: its failed recognition is no plugin's failure.
    await expect(page.locator('.task-popover')).not.toContainText(PLUGIN_FAILURE)
    expect(failedFetches).toStrictEqual([])
  })
})
