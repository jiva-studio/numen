import { test, expect } from 'playwright/test'

test.describe('Notes and File Tree', () => {
  test('lists vault notes and expands directory tree', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Click on the Files tab in the right panel
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Verify root items from test vault are listed
    await expect(page.getByText('Thermodynamics.md').first()).toBeVisible()
    await expect(page.getByText('notes').first()).toBeVisible()
    await expect(page.getByText('assets').first()).toBeVisible()

    // Target notes folder row in the file tree and expand it with Enter
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Verify Entropy.md appears inside expanded notes folder
    await expect(page.getByText('Entropy.md').first()).toBeVisible()
  })
})
