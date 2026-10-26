import { test, expect } from 'playwright/test'

test.describe('Window Controls and Display Preferences', () => {
  test('closes active tab cleanly via close command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 15_000,
    })

    // 1. Open notes and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    const entropyTab = page.getByRole('tab', { name: /Entropy/i }).first()
    await expect(entropyTab).toBeVisible({ timeout: 10_000 })

    // 2. Open palette and execute Close command (Close tab)
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Close tab')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Verify Entropy tab is closed
    await expect(entropyTab).not.toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('adjusts reading font size via command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 15_000,
    })

    // 1. Open palette and search "Reading font size"
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Reading font size')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 2. Selection options are presented
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await page.keyboard.press('ArrowDown')
    await page.keyboard.press('Enter')

    // 3. Verify palette closes cleanly upon choice selection
    await expect(paletteInput).not.toBeVisible({ timeout: 5_000 })
    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('toggles plexus heading nesting preference via command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 15_000,
    })

    // 1. Open palette and search "Nest headings under notes"
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Nest headings under notes')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 2. Select first offered setting value
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await page.keyboard.press('ArrowDown')
    await page.keyboard.press('Enter')

    // 3. Verify palette closes cleanly and workspace remains stable
    await expect(paletteInput).not.toBeVisible({ timeout: 5_000 })
    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 10_000,
    })
  })
})
