import { test, expect } from 'playwright/test'

test.describe('File System Operations and Organization', () => {
  test('navigates and expands folders in file tree', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand cards folder
    const cardsRow = page.locator('[data-tree-row="cards"]')
    await expect(cardsRow).toBeVisible()
    await cardsRow.press('Enter')
    await expect(page.locator('[data-tree-row="cards/Animals.md"]')).toBeVisible()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')
    await expect(page.locator('[data-tree-row="assets/sample.pdf"]')).toBeVisible()
  })

  test('triggers context menu on directory tree row', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()

    // Right-click to open row context menu
    await notesRow.click({ button: 'right' })

    // Verify context menu is displayed with file operations
    await expect(page.getByRole('menu').or(page.locator('[role="menu"]')).first()).toBeVisible({
      timeout: 5_000,
    })
  })

  test('inspects file drop targets on directory rows', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Verify directory rows define drop targets for organization
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await expect(assetsRow).toHaveAttribute('data-file-drop-target', 'assets')

    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await expect(notesRow).toHaveAttribute('data-file-drop-target', 'notes')
  })
})
