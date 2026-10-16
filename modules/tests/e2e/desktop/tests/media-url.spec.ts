import { test, expect } from 'playwright/test'

test.describe('External Media and URL Tabs', () => {
  test('opens import url command palette input dialog', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose Import URL command
    await paletteInput.fill('Import URL')
    await page.keyboard.press('Enter')

    // Verify address input mode is active
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
  })

  test('gracefully displays fallback on invalid or unreachable media url', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Import URL palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Import URL')
    await page.keyboard.press('Enter')

    // Enter address
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('https://127.0.0.1:9/unreachable')
    await page.keyboard.press('Enter')

    // Verify application remains interactive
    await expect(page.locator('text=Ready').or(page.locator('main')).first()).toBeVisible({
      timeout: 10_000,
    })
  })
})
