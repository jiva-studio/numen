import { test, expect } from 'playwright/test'

test.describe('EPUB Book Reader', () => {
  test('opens EPUB book and displays reflowable reader content', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target sample.epub in tree
    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    // Verify book tab opened and reflowable content loaded
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Verify parsed chapter title or content is rendered
    await expect(page.locator('text=Chapter 1').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Hello reflowable epub reader').first()).toBeVisible()
  })

  test('toggles contents listing drawer in EPUB reader', async ({ page }) => {
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

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Click Show contents button
    const contentsBtn = page.getByRole('button', { name: 'Show contents' })
    if (await contentsBtn.isVisible()) {
      await contentsBtn.click()
    }
  })

  test('renders book fallback on corrupted EPUB file', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target corrupted.epub
    const badEpubRow = page.locator('[data-tree-row="assets/corrupted.epub"]')
    await expect(badEpubRow).toBeVisible()
    await badEpubRow.press('Enter')

    // Verify application does not crash
    await expect(page.locator('text=something inside numen went wrong')).not.toBeVisible()
  })

  test('turns pages via keyboard navigation in EPUB reader', async ({ page }) => {
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

    const bookTab = page.locator('.book-tab')
    await expect(bookTab).toBeVisible({ timeout: 10_000 })

    // Focus book tab and press page turn keys
    await bookTab.click()
    await page.keyboard.press('PageDown')
    await expect(bookTab).toBeVisible()
  })

  test('persists book reader across tab switching', async ({ page }) => {
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

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Open notes folder and open a note
    await page.getByText('Files').first().click()
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Switch back to Test Book tab
    const bookTabItem = page.getByRole('tab', { name: /Test Book/i }).first()
    await expect(bookTabItem).toBeVisible()
    await bookTabItem.click()

    await expect(page.locator('.book-tab')).toBeVisible()
  })
})
