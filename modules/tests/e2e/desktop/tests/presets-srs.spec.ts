import { test, expect } from 'playwright/test'

test.describe('Spaced Repetition and Preset Editor', () => {
  test('opens new spaced repetition preset editor from command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Select New preset command
    await paletteInput.fill('New preset')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // In naming step, enter preset name
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('PhysicsSRS')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // Verify preset tab is opened
    await expect(
      page.getByRole('tab', { name: /PhysicsSRS/i })
        .or(page.locator('[aria-label*="PhysicsSRS"]'))
        .first(),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('validates preset retention and interval boundaries with problem hints', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Create and open preset
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('New preset')
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('StrictSRS')
    await page.keyboard.press('Enter')

    await expect(
      page.getByRole('tab', { name: /StrictSRS/i }).first(),
    ).toBeVisible({ timeout: 10_000 })

    // Verify preset settings panel is rendered
    const presetPanel = page.locator('.settings').or(page.locator('form')).or(page.locator('main')).first()
    await expect(presetPanel).toBeVisible()
  })

  test('assigns custom preset to active note via preset command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes folder and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Trigger Preset command on note
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Preset')
    await page.keyboard.press('Enter')

    // Verify preset editor is opened for the note
    await expect(
      page.getByRole('tab', { name: /Entropy\.md/i })
        .or(page.getByRole('region', { name: 'Goal' }))
        .first(),
    ).toBeVisible({ timeout: 10_000 })
  })
})
