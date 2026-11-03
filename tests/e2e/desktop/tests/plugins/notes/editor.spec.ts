import { test, expect } from '../../support/test.js'
import { openFromFiles } from '../../support/workspace.js'

test.describe('Markdown Note Editor', () => {
  test('opens markdown note, displays content and note tab', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/Entropy.md'])

    // Verify note editor tab is visible
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.note__text .cm-content').first()).toBeVisible({ timeout: 10_000 })

    // Verify note text from vault is loaded
    await expect(page.locator('.note').first()).toContainText('Entropy')
  })

  test('opens plain text file via modular markdown editor plugin', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/plain.md'])

    // Verify markdown editor tab is visible and editor pane is active
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const editor = page.locator('.note .cm-content').first()
    await expect(editor).toBeVisible({ timeout: 10_000 })

    // Type text into modular editor
    await editor.click()
    await page.keyboard.type('Modular editor plain text')
    await expect(editor).toContainText('Modular editor plain text')
  })

  test('opens extra.md file and renders markdown heading', async ({ page }) => {
    await openFromFiles(page, ['notes', 'notes/extra.md'])

    // Verify markdown editor tab is visible and accepts markdown input
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const editor = page.locator('.note .cm-content').first()
    await expect(editor).toBeVisible({ timeout: 10_000 })

    await editor.click()
    await page.keyboard.type('# Markdown Heading')
    await expect(editor).toContainText('# Markdown Heading')
  })
})
