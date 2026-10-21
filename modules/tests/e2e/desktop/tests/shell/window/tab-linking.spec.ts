import { test, expect } from 'playwright/test'

test.describe('Tab Linking and Split Pane Coordination', () => {
  test('opens note beside current reader tab via beside command', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open notes folder and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Open palette and choose Beside command
    await page.keyboard.press('Control+p')
    const paletteInput = page.locator('input[data-palette="field"]').first()
    await expect(paletteInput).toBeVisible({ timeout: 5_000 })

    await paletteInput.fill('Beside')
    await page.keyboard.press('Enter')

    // 3. Verify split separator is active
    await expect(page.getByRole('separator', { name: 'Resize panes' })).toBeVisible({
      timeout: 10_000,
    })
  })

  test('links tabs across split panes into coordinated group', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open sample.pdf
    await page.getByText('Files').first().click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const pdfRow = page.locator('[data-tree-row="assets/sample.pdf"]')
    await expect(pdfRow).toBeVisible()
    await pdfRow.press('Enter')

    // Open notes and Entropy.md
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    // Verify both tab headers exist in tablists
    await expect(page.getByRole('tab', { name: /sample\.pdf/i }).first()).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByRole('tab', { name: /Entropy/i }).first()).toBeVisible({
      timeout: 10_000,
    })
  })

  test('cleans up tab link group when one linked tab is closed', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open sample.epub
    await page.getByText('Files').first().click()
    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    if (!(await epubRow.isVisible())) {
      const assetsRow = page.locator('[data-tree-row="assets"]')
      await expect(assetsRow).toBeVisible()
      await assetsRow.press('Enter')
    }

    await expect(epubRow).toBeVisible({ timeout: 5_000 })
    await epubRow.press('Enter')

    const epubTab = page.getByRole('tab', { name: /Test Book|sample\.epub/i }).first()
    await expect(epubTab).toBeVisible({ timeout: 10_000 })

    // Close epub tab
    await page.keyboard.press('Control+Shift+w')
    await expect(epubTab).not.toBeVisible({ timeout: 5_000 })
  })
})
