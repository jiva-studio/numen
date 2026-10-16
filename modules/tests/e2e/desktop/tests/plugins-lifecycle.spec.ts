import { test, expect } from 'playwright/test'

test.describe('Plugin Lifecycle and Process Management', () => {
  test('activates reader-pdf plugin on demand and cleans up on close', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.pdf to trigger on-demand activation
    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    // Verify reader rendered
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Close the tab
    const closeBtn = page.getByRole('button', { name: /Close sample\.pdf/i }).first()
    if (await closeBtn.isVisible()) {
      await closeBtn.click()
      await expect(page.locator('.reader')).not.toBeVisible({ timeout: 5_000 })
    }
  })

  test('activates reader-epub plugin on demand and cleans up on close', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Open sample.epub
    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    // Verify book tab rendered
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Close the tab
    const closeBtn = page.getByRole('button', { name: /Close Test Book/i }).first()
    if (await closeBtn.isVisible()) {
      await closeBtn.click()
      await expect(page.locator('.book-tab')).not.toBeVisible({ timeout: 5_000 })
    }
  })
})
