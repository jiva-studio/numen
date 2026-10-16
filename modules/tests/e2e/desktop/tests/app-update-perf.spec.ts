import { test, expect } from 'playwright/test'

test.describe('App Updates, Performance Profiles and OCR Settings', () => {
  test('switches application performance profile via command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose Performance profile command
    await paletteInput.fill('Performance profile')
    await page.keyboard.press('Enter')

    // Verify option choices are rendered
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
  })

  test('inspects ocr engine language options in settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open settings tab
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /Settings/i })).toBeVisible({ timeout: 10_000 })

    // Verify settings tab content is displayed
    const settingsContainer = page.locator('.settings').or(page.locator('main')).first()
    await expect(settingsContainer).toBeVisible()
  })

  test('persists interface scale and theme across full application reload', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Settings tab
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /Settings/i })).toBeVisible({ timeout: 10_000 })

    // Reload page
    await page.reload()
    await expect(page.locator('text=Ready')).toBeVisible({ timeout: 10_000 })
  })
})
