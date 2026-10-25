import { test, expect } from 'playwright/test'
import { getGraph, openWorkspace } from '../../support/workspace'

/** The commands a keystroke holding Shift reaches, pressed with the palette nowhere in sight. */
test.describe('a command reached by a keystroke holding Shift', () => {
  test('opens an agent in a tab of its own', async ({ page }) => {
    await openWorkspace(page)
    const agents = page.getByRole('tab', { name: /Agent/ })
    const before = await agents.count()

    await page.keyboard.press('Control+Shift+a')

    await expect(agents).toHaveCount(before + 1)
  })

  test('shows the note in front in the plex', async ({ page }) => {
    await openWorkspace(page)
    await page.getByText('Files').first().click()
    await page.locator('[data-tree-row="notes"]').press('Enter')
    await page.locator('[data-tree-row="notes/Entropy.md"]').press('Enter')
    await expect(page.locator('.note').first()).toBeVisible()

    await page.keyboard.press('Control+Shift+x')

    await expect(getGraph(page).getByText('Entropy').first()).toBeVisible()
  })

  test('draws every keystroke that holds Shift on the row that names it', async ({ page }) => {
    await openWorkspace(page)
    await page.keyboard.press('Control+p')
    await expect(page.locator('input[data-palette="field"]').first()).toBeVisible()

    for (const letter of ['X', 'A', 'W']) {
      await expect(
        page
          .locator('[data-palette="list"]')
          .getByText(new RegExp(`⇧\\s*${letter}|Shift\\s*\\+?\\s*${letter}`, 'i'))
          .first(),
      ).toBeVisible()
    }
  })
})
