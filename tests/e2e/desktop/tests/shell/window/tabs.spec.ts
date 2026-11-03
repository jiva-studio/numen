import { test, expect } from '../../support/test.js'
import { openFromFiles, openTreeRow } from '../../support/workspace.js'

test.describe('Workspace Tab Management', () => {
  test('opens multiple tabs and supports tab switching', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/Entropy.md'])

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Open Files panel again to open an EPUB book
    await page.getByRole('tab', { name: /^Files/ }).click()

    await openTreeRow(page, 'assets')

    await openTreeRow(page, 'assets/sample.epub')

    await expect(page.locator('.book-tab').first()).toBeVisible({ timeout: 10_000 })

    // 3. Verify multiple tabs exist in the workspace
    const tabs = page.locator('.tab')
    await expect(tabs.first()).toBeVisible()

    // 4. Switch back to Entropy note tab and verify content
    const entropyTab = page.getByRole('tab', { name: /Entropy/i })
    await expect(entropyTab).toBeVisible()
    await entropyTab.click()
    await expect(page.locator('.note').first()).toBeVisible()
  })
})
