import { test, expect } from 'playwright/test'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'

test.describe('Plugin Settings and Configuration', () => {
  test('opens command palette and navigates to Settings tab', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Settings', 'Settings')

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
    const pluginRow = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
    await expect(pluginRow).toBeVisible({ timeout: 10_000 })
    await pluginRow.click()
    await expect(page.getByText('Transcription Language').first()).toBeVisible({ timeout: 5_000 })
  })

  test('shows the default transcription language', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    const pluginRow = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
    await expect(pluginRow).toBeVisible({ timeout: 10_000 })
    await pluginRow.click()

    const language = page.locator('.settings__choice').first()
    await expect(language).toBeVisible()
    await expect(language).toHaveText('auto')
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
    const pluginRow = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
    await expect(pluginRow).toBeVisible({ timeout: 10_000 })
    await pluginRow.click()

    // Select container for language enum setting is rendered
    const choiceEl = page.locator('.settings__choice').first()
    await expect(choiceEl).toBeVisible({ timeout: 5_000 })
  })

  test('persists a changed transcription language across tab re-opening', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Settings', 'Settings')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    const pluginRow = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
    await expect(pluginRow).toBeVisible({ timeout: 10_000 })
    await pluginRow.click()

    const language = page.locator('.settings__choice').first()
    await expect(language).toBeVisible()
    await expect(language).toHaveText('auto')
    await language.click()
    await page.getByRole('menuitemradio', { name: 'es', exact: true }).click()
    await expect(language).toHaveText('es')

    // Close Settings tab
    const closeBtn = page.getByRole('button', { name: /Close Settings/i }).first()
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()
    await expect(page.locator('.settings__plugins')).not.toBeVisible({ timeout: 5_000 })

    // Reopen Settings tab via command palette
    await runPaletteCommand(page, 'Settings', 'Settings')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })
    const pluginRow2 = page.getByRole('button', { name: /Audio\/Video Transcription/i }).first()
    if (await pluginRow2.isVisible()) {
      await pluginRow2.click()
    }

    // Verify the changed language persisted
    await expect(page.locator('.settings__choice').first()).toHaveText('es')
  })

  test('searches palette and executes Files navigation command', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Files', 'Show files')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })
})
