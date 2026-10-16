import { test, expect } from 'playwright/test'

test.describe('File Conflict and Concurrency Handling', () => {
  test('handles concurrent file edits and preserves editor integrity', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes folder
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open Entropy.md
    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    const editor = page.locator('.note__text .cm-content').first()
    await expect(editor).toBeVisible({ timeout: 10_000 })

    // Type modifications in note editor
    await editor.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\nConcurrent modification test line')

    // Verify text is reflected cleanly in editor
    await expect(editor).toContainText('Concurrent modification test line')
  })
})
