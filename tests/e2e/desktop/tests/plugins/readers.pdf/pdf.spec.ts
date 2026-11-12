import { test, expect } from '../../support/test.js'
import { openAsset, openTreeRow } from '../../support/workspace.js'

test.describe('PDF Document Reader', () => {
  test('opens paper.pdf document from vault assets', async ({ page }) => {
    await openAsset(page, 'paper.pdf')

    // Verify document tab opened and reader rendered
    await expect(page.locator('.fallback-panel')).toBeHidden({ timeout: 5_000 })
    await expect(page.locator('text=something inside numen went wrong')).toBeHidden()
    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // The page image arrives from the plugin's own route and decodes
    const picture = page.locator('.reader__picture').first()
    await expect(picture).toHaveAttribute('src', /^\/plugin\/md\.numen\.readers\.pdf\/pages\//, {
      timeout: 10_000,
    })
    await expect
      .poll(
        () => picture.evaluate((img: HTMLImageElement) => (img.complete ? img.naturalWidth : 0)),
        { timeout: 15_000 },
      )
      .toBeGreaterThan(0)
  })

  test('persists PDF reader across tab switching', async ({ page }) => {
    await openAsset(page, 'sample.pdf')

    await expect(page.locator('.reader')).toBeVisible({ timeout: 10_000 })

    // Open notes folder and open a note
    await page.getByRole('tab', { name: /^Files/ }).click()
    await openTreeRow(page, 'notes')

    await openTreeRow(page, 'notes/Entropy.md')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Switch back to sample.pdf tab
    const pdfTab = page.getByRole('tab', { name: /sample\.pdf/i }).first()
    await expect(pdfTab).toBeVisible()
    await pdfTab.click()

    await expect(page.locator('.reader')).toBeVisible()
  })
})
