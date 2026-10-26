import { test, expect } from 'playwright/test'

test.describe('EPUB Book Reader Contents and Chapter Navigation', () => {
  test('toggles reflowable book table of contents listing', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // 1. Open assets folder and sample.epub
    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // 2. Locate contents toggle button in book foot
    const showContentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(showContentsBtn).toBeVisible({ timeout: 10_000 })

    // 3. Open table of contents
    await showContentsBtn.click()

    // 4. Verify contents drawer is open with navigation
    const contentsNav = page.getByRole('navigation', { name: 'Contents' })
    await expect(contentsNav).toBeVisible({ timeout: 5_000 })

    // 5. Toggle button now reflects Hide contents
    const hideContentsBtn = page.getByRole('button', { name: 'Hide contents' })
    await expect(hideContentsBtn).toBeVisible()

    // 6. Dismiss contents via Escape key
    await page.keyboard.press('Escape')
    await expect(contentsNav).not.toBeVisible({ timeout: 5_000 })
  })

  test('jumps to selected chapter from table of contents', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Open contents panel
    const showContentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(showContentsBtn).toBeVisible({ timeout: 10_000 })
    await showContentsBtn.click()

    const contentsNav = page.getByRole('navigation', { name: 'Contents' })
    await expect(contentsNav).toBeVisible({ timeout: 5_000 })

    // Locate chapter buttons or fallback in contents
    const chapterButtons = contentsNav.locator('button.contents__line')
    if ((await chapterButtons.count()) > 0) {
      const firstChapter = chapterButtons.first()
      await expect(firstChapter).toBeVisible()
      await firstChapter.click()
      // Contents drawer closes upon selecting chapter
      await expect(contentsNav).not.toBeVisible({ timeout: 5_000 })
    } else {
      await expect(contentsNav.getByText('No table of contents.')).toBeVisible()
    }
  })

  test('filters book chapters using contents search field', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    await page.getByRole('tab', { name: /^Files/ }).click()
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Open contents panel
    const showContentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(showContentsBtn).toBeVisible({ timeout: 10_000 })
    await showContentsBtn.click()

    const contentsNav = page.getByRole('navigation', { name: 'Contents' })
    await expect(contentsNav).toBeVisible({ timeout: 5_000 })

    // Locate search field
    const findField = contentsNav
      .getByRole('textbox', { name: 'Find in contents' })
      .or(contentsNav.locator('input[placeholder="Find in contents"]'))
    await expect(findField).toBeVisible()

    // Type a non-existent search query
    await findField.fill('NonExistentChapterQuery123')
    await expect(contentsNav.getByText('No table of contents.')).toBeVisible({ timeout: 5_000 })

    // Clear search
    await findField.fill('')
    await expect(findField).toHaveValue('')
  })
})
