import { test, expect } from 'playwright/test'

test.describe('Chaos, Fault Tolerance and Stress Scenarios', () => {
  test('gracefully parses malformed yaml frontmatter without editor crash', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes folder
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open malformed.md
    const malformedRow = page.locator('[data-tree-row="notes/malformed.md"]')
    await expect(malformedRow).toBeVisible()
    await malformedRow.press('Enter')

    // Verify note is rendered without crashing the application
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Ready')).toBeVisible()
  })

  test('renders high-volume document smoothly without interface freeze', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open notes folder
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open large.md
    const largeRow = page.locator('[data-tree-row="notes/large.md"]')
    await expect(largeRow).toBeVisible()
    await largeRow.press('Enter')

    // Verify large document editor is rendered
    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })
    const content = page.locator('.note__text .cm-content').first()
    await expect(content).toBeVisible({ timeout: 10_000 })
  })

  test('handles rapid tab switching and closing across heterogeneous file types', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Files panel
    await page.getByText('Files').first().click()

    // Open notes folder
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    // Open Entropy.md
    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    // Switch back to Files tab to open assets
    await page.getByText('Files').first().click()

    // Open assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.pdf
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    // Rapid switch between tabs
    const entropyTab = page.getByRole('tab', { name: /Entropy/i }).first()
    const pdfTab = page.getByRole('tab', { name: /sample\.pdf/i }).first()

    await expect(entropyTab).toBeVisible({ timeout: 10_000 })
    await expect(pdfTab).toBeVisible({ timeout: 10_000 })

    await entropyTab.click()
    await pdfTab.click()
    await entropyTab.click()

    // Verify workspace remains responsive
    await expect(page.locator('text=Ready')).toBeVisible()
  })
})
