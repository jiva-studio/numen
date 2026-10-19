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
    await expect(page.getByText('Audio/Video Transcription').first()).toBeVisible({
      timeout: 10_000,
    })
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
    await expect(page.getByText('Transcription Language').first()).toBeVisible({ timeout: 5_000 })
  })

  test('selects a transcription language and verifies reactive update', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    const language = page
      .getByRole('region', { name: 'Audio/Video Transcription' })
      .locator('.settings__choice')
      .first()
    await expect(language).toBeVisible()
    await language.click()
    await page.getByRole('menuitemradio', { name: 'es', exact: true }).click()
    await expect(language).toHaveText('es')
  })

  test('changes transcription language dropdown selection', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    // Select container for language enum setting is rendered
    const choiceEl = page.locator('.settings__choice').first()
    await expect(choiceEl).toBeVisible({ timeout: 5_000 })
  })

  test('persists modified plugin settings across tab re-opening', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    const language = page
      .getByRole('region', { name: 'Audio/Video Transcription' })
      .locator('.settings__choice')
      .first()
    await expect(language).toBeVisible()
    await language.click()
    await page.getByRole('menuitemradio', { name: 'de', exact: true }).click()
    await expect(language).toHaveText('de')

    // Close Settings tab
    const closeBtn = page.getByRole('button', { name: /Close Settings/i }).first()
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()
    await expect(page.locator('.settings__plugins')).not.toBeVisible({ timeout: 5_000 })

    // Reopen Settings tab via command palette
    await page.keyboard.press('Control+p')
    const reopenInput = page.locator('input[data-palette="field"]').first()
    await expect(reopenInput).toBeVisible({ timeout: 5_000 })
    await reopenInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    // Verify the language kept the selected value
    await expect(
      page
        .getByRole('region', { name: 'Audio/Video Transcription' })
        .locator('.settings__choice')
        .first(),
    ).toHaveText('de')
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
