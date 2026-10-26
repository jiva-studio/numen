import { test, expect } from 'playwright/test'

test.describe('Stencil Templates and Schema Editor', () => {
  test('opens stencil template file and displays schema content', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand cards folder
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    // Open Term.md
    const termRow = page.locator('[data-tree-row="cards/Term.md"]')
    await expect(termRow).toBeVisible()
    await termRow.press('Enter')

    // Verify card / stencil view is rendered
    await expect(
      page
        .getByRole('group', { name: 'Stencil' })
        .or(page.getByRole('region', { name: 'Fields' }))
        .or(page.locator('.note'))
        .or(page.locator('.deck'))
        .first(),
    ).toBeVisible({ timeout: 10_000 })
  })

  test('initiates new stencil creation via command palette', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose New stencil command
    await paletteInput.fill('New stencil')
    await page.keyboard.press('Enter')

    // Verify naming step is reached
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
  })
})
