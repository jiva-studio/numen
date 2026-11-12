import { test, expect } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'

/** The commands triggered by a keystroke holding Shift, pressed with the palette closed. */
test.describe('a command reached by a keystroke holding Shift', () => {
  test('shows the agent tab that is open', async ({ page }) => {
    await openWorkspace(page)
    const agents = page.getByRole('tab', { name: /Agent/ })
    await page.getByRole('tab', { name: /^Files/ }).click()
    await expect(agents).toHaveAttribute('aria-selected', 'false')
    const before = await agents.count()

    await page.keyboard.press('Control+Shift+a')

    await expect(agents.first()).toHaveAttribute('aria-selected', 'true')
    await expect(agents).toHaveCount(before)
  })

  test('shows the note in front in the plex', async ({ page }) => {
    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.locator('[data-tree-row="notes"]').press('Enter')
    await page.locator('[data-tree-row="notes/Entropy.md"]').press('Enter')
    await expect(page.locator('.note').first()).toBeVisible()

    await page.keyboard.press('Control+Shift+x')

    await expect(getGraph(page).getByText('Entropy').first()).toBeVisible()
  })
})
