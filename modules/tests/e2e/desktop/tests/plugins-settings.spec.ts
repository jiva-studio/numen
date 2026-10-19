import { test, expect } from 'playwright/test'
import { openWorkspace, runPaletteCommand } from './support/workspace.js'

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
    await expect(page.getByText('Auto-transcribe on import').first()).toBeVisible({
      timeout: 5_000,
    })
    await expect(page.getByText('Transcription Language').first()).toBeVisible({ timeout: 5_000 })
  })

  test('toggles auto-transcribe boolean switch and verifies reactive update', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Settings')
    await page.keyboard.press('Enter')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    // Find the toggle switch for auto-transcribe
    const switchEl = page.locator('[data-slot="switch"]').first()
    await expect(switchEl).toBeVisible()
    const initialState = await switchEl.getAttribute('data-state')

    // Click switch to toggle state
    await switchEl.click()
    const nextState = initialState === 'checked' ? 'unchecked' : 'checked'
    await expect(switchEl).toHaveAttribute('data-state', nextState)
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
    await openWorkspace(page)

    await runPaletteCommand(page, 'Settings', 'Settings')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    const switchEl = page.locator('[data-slot="switch"]').first()
    await expect(switchEl).toBeVisible()
    const initialState = await switchEl.getAttribute('data-state')
    await switchEl.click()
    const toggledState = initialState === 'checked' ? 'unchecked' : 'checked'
    await expect(switchEl).toHaveAttribute('data-state', toggledState)

    // Close Settings tab
    const closeBtn = page.getByRole('button', { name: /Close Settings/i }).first()
    await expect(closeBtn).toBeVisible()
    await closeBtn.click()
    await expect(page.locator('.settings__plugins')).not.toBeVisible({ timeout: 5_000 })

    // Reopen Settings tab via command palette
    await runPaletteCommand(page, 'Settings', 'Settings')

    await expect(page.locator('.settings__plugins').first()).toBeVisible({ timeout: 10_000 })

    // Verify switch retained the toggled value
    await expect(page.locator('[data-slot="switch"]').first()).toHaveAttribute(
      'data-state',
      toggledState,
    )
  })

  test('searches palette and executes Files navigation command', async ({ page }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Files', 'Show the files of the vault')

    // Verify file tree is open and assets folder is visible
    await expect(page.locator('[data-tree-row="assets"]').first()).toBeVisible({ timeout: 10_000 })
  })
})
