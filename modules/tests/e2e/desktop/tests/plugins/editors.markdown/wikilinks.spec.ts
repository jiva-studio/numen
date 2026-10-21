import { test, expect } from 'playwright/test'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'

test.describe('Note Creation and Wikilinks', () => {
  test('creates new note via command palette naming flow', async ({ page }) => {
    await openWorkspace(page)

    // Choose New note command
    await runPaletteCommand(page, 'New note', 'New note')

    // The naming step is drawn, and the name is offered back once typed
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    await page.locator('input[data-palette="field"]').first().fill('Quantum')
    await expect(page.locator('[role="option"][aria-selected="true"]')).toContainText('Quantum')
    await page.keyboard.press('Enter')

    // Verify new note tab opened and is the one shown
    await expect(page.getByRole('tab', { name: /Quantum/i }).first()).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  test('renders wikilink syntax in markdown editor', async ({ page }) => {
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

    const noteEditor = page.locator('.note__text .cm-content').first()
    await expect(noteEditor).toBeVisible({ timeout: 10_000 })

    // Type a wikilink
    await noteEditor.click()
    await page.keyboard.press('End')
    await page.keyboard.type('\n[[Thermodynamics]]')

    // Verify wikilink text is rendered in CodeMirror
    await expect(noteEditor).toContainText('Thermodynamics')
  })

  test('navigates to target note upon clicking wikilink', async ({ page }) => {
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

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Open Thermodynamics tab from neighbourhood / link
    const linkBtn = page
      .locator('.plex')
      .getByRole('button', { name: /Thermodynamics/i })
      .or(page.locator('.plex .plex__title', { hasText: 'Thermodynamics' }))
      .first()
    await expect(linkBtn).toBeVisible({ timeout: 10_000 })
    await linkBtn.dblclick({ force: true })

    await expect(page.getByRole('tab', { name: /Thermodynamics/i }).first()).toBeVisible({
      timeout: 10_000,
    })
  })
})
