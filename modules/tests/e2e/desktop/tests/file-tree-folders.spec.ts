import { test, expect } from 'playwright/test'

test.describe('File Tree Folder Expansion and Structure', () => {
  test('creates new nested folder via file tree context menu', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open Files tab
    await page.getByText('Files').first().click()

    // 2. Right click on notes folder row to open context menu
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.click({ button: 'right' })

    // 3. Click New folder option in menu
    const newFolderItem = page
      .getByRole('menuitem', { name: /New folder/i })
      .or(page.getByText('New folder'))
      .first()
    await expect(newFolderItem).toBeVisible({ timeout: 5_000 })
    await newFolderItem.click()

    // 4. Verify new folder was created and appears under notes
    const createdFolderRow = page.locator('[data-tree-row^="notes/New folder"]').first()
    await expect(createdFolderRow).toBeVisible({ timeout: 10_000 })
  })

  test('persists folder expansion state across navigation', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open Files tab and expand cards directory
    await page.getByText('Files').first().click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    const animalsRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(animalsRow).toBeVisible({ timeout: 5_000 })
    await expect(cardsRow).toHaveAttribute('aria-expanded', 'true')

    // 2. Switch to another tab (e.g. Thermodynamics or Agent tab)
    const otherTab = page.getByRole('tab', { name: /Thermodynamics|Agent/i }).first()
    await otherTab.click()

    // 3. Switch back to Files tab
    await page.getByText('Files').first().click()

    // 4. Verify cards folder remains expanded
    await expect(cardsRow).toBeVisible()
    await expect(cardsRow).toHaveAttribute('aria-expanded', 'true')
    await expect(animalsRow).toBeVisible()
  })

  test('collapses expanded folder on Enter key', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open Files tab and expand cards directory
    await page.getByText('Files').first().click()
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')

    const animalsRow = page.locator('[data-tree-row="cards/Animals.md"]')
    await expect(animalsRow).toBeVisible({ timeout: 5_000 })
    await expect(cardsRow).toHaveAttribute('aria-expanded', 'true')

    // 2. Press Enter again on cards folder to collapse
    await cardsRow.press('Enter')

    // 3. Verify cards folder is collapsed and child file is no longer visible
    await expect(cardsRow).toHaveAttribute('aria-expanded', 'false')
    await expect(animalsRow).not.toBeVisible()
  })
})
