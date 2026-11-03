import { test, expect } from '../../support/test.js'
import { openAsset, openTreeRow } from '../../support/workspace.js'

test.describe('EPUB Book Reader', () => {
  test('opens EPUB book and displays reflowable reader content', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    // Verify book tab opened and reflowable content loaded
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Verify parsed chapter title or content is rendered
    await expect(page.locator('text=Chapter 1').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Hello reflowable epub reader').first()).toBeVisible()
  })

  test('toggles contents listing drawer in EPUB reader', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Click Show contents button
    const contentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(contentsBtn).toBeVisible()
    await contentsBtn.click()
    await expect(page.getByRole('group', { name: 'Contents' })).toBeVisible()
  })

  test('renders book fallback on corrupted EPUB file', async ({ page }) => {
    await openAsset(page, 'corrupted.epub')

    // Verify application does not crash
    await expect(page.locator('text=something inside numen went wrong')).toBeHidden()
  })

  test('turns pages via keyboard navigation in EPUB reader', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    const bookTab = page.locator('.book-tab')
    await expect(bookTab).toBeVisible({ timeout: 10_000 })

    // The book tab takes the page turn keys once it holds the focus. A click is refused in
    // WebKit, where elementsFromPoint answers <html> over the book, so the tab is focused.
    await bookTab.focus()
    await expect(bookTab).toBeFocused()
    await page.keyboard.press('PageDown')
    await expect(bookTab).toBeVisible()
  })

  test('persists book reader across tab switching', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Open notes folder and open a note
    await page.getByRole('tab', { name: /^Files/ }).click()
    await openTreeRow(page, 'notes')

    await openTreeRow(page, 'notes/Entropy.md')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // Switch back to Test Book tab
    const bookTabItem = page.getByRole('tab', { name: /Test Book|sample\.epub/i }).first()
    await expect(bookTabItem).toBeVisible()
    await bookTabItem.click()

    await expect(page.locator('.book-tab')).toBeVisible()
  })
})
