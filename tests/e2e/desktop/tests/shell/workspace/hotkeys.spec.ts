import { test, expect } from '../../support/test.js'
import { openAsset } from '../../support/workspace.js'

test.describe('Keyboard Shortcuts and Navigation', () => {

  test('closes active plugin tab via Control+Shift+w keyboard shortcut', async ({ page }) => {
    await openAsset(page, 'sample.pdf')

    const reader = page.locator('.reader')
    await expect(reader).toBeVisible({ timeout: 10_000 })

    // Press Control+Shift+w (Numen chord for close tab)
    await page.keyboard.press('Control+Shift+w')

    // Verify reader tab was closed
    await expect(reader).not.toBeVisible({ timeout: 5_000 })
  })
})
