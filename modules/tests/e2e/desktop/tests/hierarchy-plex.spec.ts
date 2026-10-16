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
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 3. Name child note
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Maxwell')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    // 4. Verify child tab is created and relationship is visible
    const maxwellTab = page.getByRole('tab', { name: /Maxwell/i }).first()
    await expect(maxwellTab).toBeVisible({ timeout: 10_000 })

    // Verify parent connection is reflected in neighbourhood
    const parentButton = page.getByRole('button', { name: /Entropy/i }).first()
    await expect(parentButton).toBeVisible({ timeout: 10_000 })
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

    // Create child note Boltzmann
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Child')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Boltzmann')
    await page.waitForTimeout(300)
    await page.keyboard.press('Enter')

    await expect(page.getByRole('tab', { name: /Boltzmann/i }).first()).toBeVisible({ timeout: 10_000 })

    // Click parent button Entropy to navigate back
    const parentLink = page.getByRole('button', { name: /Entropy/i }).first()
    await expect(parentLink).toBeVisible({ timeout: 10_000 })
    await parentLink.click()

    // Verify Entropy tab is active and visible
    await expect(page.getByRole('tab', { name: /Entropy/i }).first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
  })

  test('interacts with plex graph node to focus related note tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Verify neighbourhood graph is visible
    const neighbourhood = page.getByRole('group', { name: 'Neighbourhood' })
    await expect(neighbourhood).toBeVisible({ timeout: 10_000 })

    // Find and click child node Entropy
    const childNode = page.getByRole('button', { name: /Entropy, child/i })
    await expect(childNode).toBeVisible({ timeout: 10_000 })
    await childNode.click()

    // Verify neighbourhood refocused to Entropy
    const entropyFocus = page.locator('[aria-label="Entropy, focus"]').or(page.getByAltText('Entropy, focus'))
    await expect(entropyFocus).toBeVisible({ timeout: 10_000 })
  })
})
