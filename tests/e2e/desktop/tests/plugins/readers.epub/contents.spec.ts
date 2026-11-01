import { test, expect } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'
import {
  createEpub,
  deleteVaultFiles,
  waitForListedFile,
  writeVaultBytes,
} from '../../support/media.js'
import { createUniqueTitle } from '../../support/vault.js'

test.describe('EPUB Book Reader Contents and Chapter Navigation', () => {
  const made: string[] = []

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

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
    await expect(contentsNav).toBeHidden({ timeout: 5_000 })
  })

  test('jumps to the chapter selected in the contents, and closes the contents', async ({
    page,
    request,
  }) => {
    const name = `${createUniqueTitle('chapters').replace(/ /g, '-')}.epub`
    made.push(`assets/${name}`)
    writeVaultBytes(
      `assets/${name}`,
      createEpub('Tide book', [
        { title: 'Dawn tide', text: 'The water is low at dawn. '.repeat(120) },
        { title: 'Dusk tide', text: 'The water is high at dusk. '.repeat(120) },
      ]),
    )
    await waitForListedFile(request, `assets/${name}`)
    await openWorkspace(page)
    await openVaultFile(page, 'assets', name)
    const pages = page.getByRole('region', { name: 'Pages' })
    await expect(pages.getByRole('heading', { name: 'Dawn tide' })).toBeVisible({ timeout: 10_000 })

    await page.getByRole('button', { name: 'Show contents' }).click()
    const contentsNav = page.getByRole('navigation', { name: 'Contents' })
    const chapters = contentsNav.locator('button.contents__line')
    await expect(chapters).toHaveText(['Dawn tide', 'Dusk tide'])
    await chapters.nth(1).click()

    await expect(contentsNav).toBeHidden()
    await expect(pages.getByRole('heading', { name: 'Dusk tide' })).toBeVisible()
  })

  test('lists the documents of a book with no table of contents by their names, and opens one', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'sample.epub')
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    await page.getByRole('button', { name: 'Show contents' }).click()

    const contentsNav = page.getByRole('navigation', { name: 'Contents' })
    const documents = contentsNav.locator('button.contents__line')
    await expect(documents).toHaveText(['doc1'])
    await documents.first().click()
    await expect(contentsNav).toBeHidden()
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
    const findField = contentsNav.getByRole('textbox', { name: 'Find in contents' })
    await expect(findField).toBeVisible()

    // Type a non-existent search query
    await findField.fill('NonExistentChapterQuery123')
    await expect(contentsNav.getByText('No table of contents.')).toBeVisible({ timeout: 5_000 })

    // Clear search
    await findField.fill('')
    await expect(findField).toHaveValue('')
  })
})
