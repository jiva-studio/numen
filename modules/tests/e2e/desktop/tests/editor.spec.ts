import { test, expect } from 'playwright/test'

test.describe('Markdown Note Editor', () => {
  test('opens markdown note, displays content and note tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand notes folder
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open Entropy.md
    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    // Verify note editor tab is visible
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note__text .cm-content').first()).toBeVisible({ timeout: 10_000 })

    // Verify note text from vault is loaded
    await expect(page.locator('.note').first()).toContainText('Entropy')
  })
})
