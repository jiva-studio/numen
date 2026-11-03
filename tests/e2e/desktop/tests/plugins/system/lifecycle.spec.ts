import { test, expect } from '../../support/test.js'
import { closeTab, openAsset, openTreeRow } from '../../support/workspace.js'

test.describe('Plugin Lifecycle and Process Management', () => {
  test('activates reader-pdf plugin on demand and cleans up on close', async ({ page }) => {
    await openAsset(page, 'sample.pdf')

    // Verify reader rendered
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Close the tab
    await closeTab(page, /Close sample\.pdf/i)
    await expect(page.locator('.reader')).toBeHidden({ timeout: 5_000 })
  })

  test('activates reader-epub plugin on demand and cleans up on close', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    // Verify book tab rendered
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Close the tab
    await closeTab(page, /Close sample\.epub/i)
    await expect(page.locator('.book-tab')).toBeHidden({ timeout: 5_000 })
  })

  test('re-activates plugin on second file opening after tab closure', async ({ page }) => {
    await openAsset(page, 'sample.pdf')
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    await closeTab(page, /Close sample\.pdf/i)
    await expect(page.locator('.reader')).toBeHidden()

    // 2. Open paper.pdf to verify plugin re-activation
    await openTreeRow(page, 'assets/paper.pdf')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })
  })

  test('manages multiple distinct plugin processes concurrently', async ({ page }) => {
    await openAsset(page, 'sample.pdf')
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Open sample.epub
    await page.getByRole('tab', { name: /^Files/ }).click()
    await openTreeRow(page, 'assets/sample.epub')
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Switch back to sample.pdf tab
    const pdfTab = page.getByRole('tab', { name: /sample\.pdf/i }).first()
    await expect(pdfTab).toBeVisible()
    await pdfTab.click()
    await expect(page.locator('.reader')).toBeVisible()
  })
})
