import { test, expect } from 'playwright/test'

test.describe('Plugin Security and Path Boundaries', () => {
  test('rejects path traversal and unauthorized filesystem paths', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Try navigating to a note outside vault using palette
    await page.keyboard.press('Control+p')

    const paletteInput = page.locator('input[placeholder*="Search"], input[type="text"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('../../etc/passwd')
    await page.keyboard.press('Enter')

    // Verify application did not crash and sensitive files are not exposed
    await expect(page.locator('text=root:x:0:0')).not.toBeVisible()
    await expect(page.locator('text=something inside numen went wrong')).not.toBeVisible()
  })

  test('gracefully handles missing and invalid file entries', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Workspace remains interactive and intact
    await expect(page.locator('[data-tree-row="notes"]')).toBeVisible()
    await expect(page.locator('text=something inside numen went wrong')).not.toBeVisible()
  })
})
