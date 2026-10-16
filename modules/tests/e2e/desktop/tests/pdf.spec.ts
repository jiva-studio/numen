import { test, expect } from 'playwright/test'

test.describe('PDF Document Reader', () => {
  test('opens PDF document and renders viewer without saving error', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target sample.pdf in tree
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    // Verify document tab opened and no error modal or error panel is shown
    await expect(page.locator('.fallback-panel')).not.toBeVisible({ timeout: 5_000 })
    await expect(page.locator('text=Saving went wrong')).not.toBeVisible()
    await expect(page.locator('text=Something went wrong')).not.toBeVisible()

    // Verify reader canvas / page rendered
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })
  })

  test('opens paper.pdf document from vault assets', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target paper.pdf in tree
    const paperRow = page.locator('[data-tree-row="assets/paper.pdf"]')
    await expect(paperRow).toBeVisible()
    await paperRow.press('Enter')

    // Verify document tab opened and reader rendered
    await expect(page.locator('.fallback-panel')).not.toBeVisible({ timeout: 5_000 })
    await expect(page.locator('text=something inside numen went wrong')).not.toBeVisible()
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })
  })

  test('handles viewport resize on PDF reader', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.pdf
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Resize viewport to trigger layout measure
    await page.setViewportSize({ width: 900, height: 600 })
    await expect(page.locator('.reader')).toBeVisible()
  })

  test('displays graceful fallback panel on corrupted PDF', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target corrupted.pdf in tree
    const badPdfRow = page.locator('[data-tree-row="assets/corrupted.pdf"]')
    await expect(badPdfRow).toBeVisible()
    await badPdfRow.press('Enter')

    // Verify fallback panel is shown gracefully displaying error message
    await expect(page.locator('.fallback-panel')).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.fallback-message')).toBeVisible()
  })

  test('navigates across pages in PDF document', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.pdf
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    const reader = page.locator('.reader')
    await expect(reader).toBeVisible({ timeout: 10_000 })

    // Focus reader and send page navigation keys
    await reader.click()
    await page.keyboard.press('PageDown')
    await expect(reader).toBeVisible()
  })

  test('persists PDF reader across tab switching', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.pdf
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Open notes folder and open a note
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Switch back to sample.pdf tab
    const pdfTab = page.getByRole('tab', { name: /sample\.pdf/i }).first()
    await expect(pdfTab).toBeVisible()
    await pdfTab.click()

    await expect(page.locator('.reader')).toBeVisible()
  })
})
