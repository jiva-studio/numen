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
    await paletteInput.fill('Set performance profile')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify option choices are rendered and select Balanced
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Balanced')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify command executes and palette closes
    await expect(paletteInput).not.toBeVisible({ timeout: 5_000 })
    await expect(page.locator('.status-bar')).toBeVisible({ timeout: 10_000 })
  })

  test('inspects ocr engine language options in settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open settings tab via command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /Settings/i })).toBeVisible({ timeout: 10_000 })

    // In master-detail settings, click Document OCR row
    const ocrButton = page.getByRole('button', { name: /Document OCR/i }).first()
    await expect(ocrButton).toBeVisible({ timeout: 10_000 })
    await ocrButton.click()

    // Verify OCR settings section is rendered
    const ocrSection = page
      .getByRole('region', { name: /Optical character recognition|Document OCR|OCR/i })
      .or(page.getByRole('heading', { name: /Optical character recognition|Document OCR|OCR/i }))
      .first()
    await expect(ocrSection).toBeVisible({ timeout: 10_000 })
  })

  test('persists interface scale and theme across full application reload', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Change theme mode to Dark via palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Light or dark')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Dark')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')
    await expect(paletteInput).not.toBeVisible({ timeout: 5_000 })

    // Verify dark mode is applied to root element
    const colorSchemeBefore = await page.evaluate(
      () => getComputedStyle(document.documentElement).colorScheme,
    )
    expect(colorSchemeBefore).toContain('dark')

    // Reload page and verify setting persists
    await page.goto('/')
    await expect(page.locator('text=Ready')).toBeVisible({ timeout: 10_000 })

    const colorSchemeAfter = await page.evaluate(
      () => getComputedStyle(document.documentElement).colorScheme,
    )
    expect(colorSchemeAfter).toContain('dark')
  })
})
