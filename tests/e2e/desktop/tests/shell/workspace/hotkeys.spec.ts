import { test, expect } from '../../support/test.js'
import { openAsset, waitForWindow } from '../../support/workspace.js'

test.describe('Keyboard Shortcuts and Navigation', () => {
  test('dismisses active palette modal on Escape key press', async ({ page }) => {
    await page.goto('/')

    await waitForWindow(page)

    // Open palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Press Escape to dismiss
    await page.keyboard.press('Escape')
    await expect(paletteInput).not.toBeVisible({ timeout: 5_000 })
  })

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
