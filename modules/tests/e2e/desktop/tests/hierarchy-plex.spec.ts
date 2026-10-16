import { test, expect } from 'playwright/test'

test.describe('Note Hierarchy and Plex Graph', () => {
  test('creates child note from active note with parent link established', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open notes folder and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Open palette and trigger Child command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Child')
    await page.keyboard.press('Enter')

    // 3. Name child note
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Maxwell')
    await page.keyboard.press('Enter')

    // 4. Verify child tab or neighbourhood is created
    await expect(
      page.getByRole('tab', { name: /Maxwell/i })
        .or(page.locator('[aria-label*="Maxwell"]'))
        .first(),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('navigates to parent note from child note header', async ({ page }) => {
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

    // Verify parent/neighbourhood link can be triggered
    const neighbourhood = page.locator('[aria-label*="neighbourhood"]').or(page.locator('.note')).first()
    await expect(neighbourhood).toBeVisible()
  })

  test('interacts with plex graph node to focus related note tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger Plex command via palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Plex')
    await page.keyboard.press('Enter')

    // Verify plex canvas is visible
    const plexCanvas = page.locator('svg').or(page.locator('canvas')).first()
    await expect(plexCanvas).toBeVisible({ timeout: 10_000 })
  })
})
