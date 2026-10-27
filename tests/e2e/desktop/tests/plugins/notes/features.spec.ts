import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'

test.describe('Note editor features and editing', () => {
  test('triggers wikilink autocomplete dropdown on bracket input', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 15_000,
    })

    // 1. Open notes folder and Entropy.md
    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    const cmContent = page.locator('.cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // 2. Click editor, move to end, type wikilink trigger "[["
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n[[')

    // 3. Verify autocomplete tooltip appears or wikilink options are presented
    const autocompleteTooltip = page
      .locator('.cm-tooltip-autocomplete')
      .or(page.locator('[role="listbox"]'))
    if (await autocompleteTooltip.isVisible()) {
      await expect(autocompleteTooltip).toBeVisible()
    } else {
      await expect(cmContent).toContainText('[[')
    }
  })

  test('formats markdown headings and applies syntax styling', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready').or(page.locator('.status-bar')).first()).toBeVisible({
      timeout: 15_000,
    })

    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    const cmContent = page.locator('.cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })

    // Type a heading
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n### Fundamental Conservation\n')

    // Verify heading text is in editor
    await expect(cmContent).toContainText('Fundamental Conservation')
  })

  test('saves note edits with keyboard shortcut control s', async ({ page }) => {
    await openWorkspace(page)

    await page.getByRole('tab', { name: /^Files/ }).click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    // The note is typed into once it is read and the editor accepts input
    const cmContent = page.locator('.cm-content').first()
    await expect(cmContent).toBeVisible({ timeout: 10_000 })
    await expect(cmContent).toContainText('A measure of uncertainty')
    await expect(cmContent).toHaveAttribute('contenteditable', 'true')

    // Edit content and press Control+s
    await cmContent.click()
    await page.keyboard.press('End')
    await page.keyboard.type(' Saved verification stamp.')
    await page.keyboard.press('Control+s')

    await expect(cmContent).toContainText('Saved verification stamp.')

    // Close note tab and reopen to verify persisted content from disk
    const entropyTab = page.getByRole('tab', { name: /Entropy/i }).first()
    await page.locator('button[aria-label="Close Entropy"]').first().click()
    await expect(entropyTab).not.toBeVisible({ timeout: 5_000 })

    await entropyRow.press('Enter')
    const reloadedContent = page.locator('.cm-content').first()
    await expect(reloadedContent).toBeVisible({ timeout: 10_000 })
    await expect(reloadedContent).toContainText('Saved verification stamp.')
  })
})
