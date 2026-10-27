import { test, expect } from '../../support/test.js'

test.describe('Markdown Note Editor', () => {
  test('opens markdown note, displays content and note tab', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

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

  test('opens plain text file via modular markdown editor plugin', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand notes folder
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open plain.md
    const txtRow = page.locator('[data-tree-row="notes/plain.md"]')
    await expect(txtRow).toBeVisible()
    await txtRow.press('Enter')

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
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByRole('tab', { name: /^Files/ }).click()

    // Expand notes folder
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open extra.md
    const mdRow = page.locator('[data-tree-row="notes/extra.md"]')
    await expect(mdRow).toBeVisible()
    await mdRow.press('Enter')

    // Verify markdown editor tab is visible and accepts markdown input
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const editor = page.locator('.note .cm-content').first()
    await expect(editor).toBeVisible({ timeout: 10_000 })

    await editor.click()
    await page.keyboard.type('# Markdown Heading')
    await expect(editor).toContainText('# Markdown Heading')
  })
})
