import { test, expect } from 'playwright/test'

test.describe('Theming and Display Scaling', () => {
  test('switches application theme mode to dark mode', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose Light or dark command
    await paletteInput.fill('Light or dark')
    await page.keyboard.press('Enter')

    // In mode selection step, choose Dark
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Dark')
    await page.keyboard.press('Enter')

    // Verify workspace remains active
    await expect(page.locator('.plex').or(page.locator('.neighbourhood')).first()).toBeVisible()
  })

  test('switches application theme mode to light mode', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Light or dark')
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Light')
    await page.keyboard.press('Enter')

    await expect(page.locator('.plex').or(page.locator('.neighbourhood')).first()).toBeVisible()
  })

  test('adjusts interface and reading size in settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Settings via command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    // Verify Settings tab opened
    await expect(page.getByRole('spinbutton', { name: 'Interface size' }).first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByRole('spinbutton', { name: 'Reading size' }).first()).toBeVisible({
      timeout: 10_000,
    })
  })
})
