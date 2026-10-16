import { test, expect } from 'playwright/test'

test.describe('Plugin Settings and Configuration', () => {
  test('opens command palette and navigates to Settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette with Control+p
    await page.keyboard.press('Control+p')

    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Search and select Settings command
    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    // Verify Settings tab opened
    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Audio/Video Transcription').first()).toBeVisible({ timeout: 10_000 })
  })

  test('displays declarative plugin settings in Settings tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Navigate to Settings
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    // Verify declarative plugin settings are rendered
    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Auto-transcribe on import').first()).toBeVisible({ timeout: 5_000 })
    await expect(page.getByText('Transcription Language').first()).toBeVisible({ timeout: 5_000 })
  })

  test('searches palette and executes Files navigation command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger palette and search Files
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Files')
    await page.keyboard.press('Enter')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })
})
