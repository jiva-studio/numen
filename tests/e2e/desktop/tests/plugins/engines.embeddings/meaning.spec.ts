import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'

test.describe('Search by meaning', () => {
  test('finds a note by a word it does not contain', async ({ page }) => {
    await openWorkspace(page)
    await page.keyboard.press('Control+k')
    const field = page.locator('input[data-palette="field"]').first()
    await expect(field).toBeVisible()

    // The note about the ferry carries its vectors once the plugin has embedded the vault.
    const passage = page
      .getByRole('group', { name: /Meaning/ })
      .getByRole('option')
      .filter({ hasText: 'The ferry leaves the harbour' })
      .first()
    await expect(async () => {
      await field.fill('')
      await field.fill('boat')
      await expect(passage).toBeVisible({ timeout: 5_000 })
    }).toPass({ timeout: 45_000 })

    await expect(
      page
        .getByRole('group', { name: /^Text/ })
        .getByRole('option')
        .filter({ hasText: 'The ferry leaves the harbour' }),
    ).toHaveCount(0)
  })
})
