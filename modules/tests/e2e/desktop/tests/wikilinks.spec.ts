import { test, expect } from 'playwright/test'

test.describe('Note Creation and Wikilinks', () => {
  test('creates new note via command palette naming flow', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger command palette
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    // Choose New note command
    await paletteInput.fill('New note')
    await page.keyboard.press('Enter')

    // In naming step, enter note name
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })
    await paletteInput.fill('Quantum')
    await page.keyboard.press('Enter')

    // Verify new note tab opened
    await expect(page.getByRole('tab', { name: /Quantum/i }).or(page.locator('.note')).first()).toBeVisible({ timeout: 10_000 })
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
    const linkBtn = page.getByRole('button', { name: /Thermodynamics/i }).or(page.getByText('Thermodynamics')).first()
    await expect(linkBtn).toBeVisible()
    await linkBtn.click()

    await expect(page.getByRole('tab', { name: /Thermodynamics/i }).first()).toBeVisible()
  })
})
