import { test, expect, type APIRequestContext, type Page } from '../../support/test.js'
import { openAsset, openVaultFile, openWorkspace, waitForPassage } from '../../support/workspace.js'
import {
  createEpub,
  deleteVaultFiles,
  waitForListedFile,
  writeVaultBytes,
} from '../../support/media.js'
import { expectSearchableList } from '../../support/searchable-list.js'
import { createUniqueTitle } from '../../support/vault.js'

test.describe('EPUB Book Reader Contents and Chapter Navigation', () => {
  const made: string[] = []

  /** Writes a book of two chapters, Dawn tide and Dusk tide, opens it and waits for the first. */
  const openTideBook = async (page: Page, request: APIRequestContext, title: string) => {
    const name = `${createUniqueTitle(title).replace(/ /g, '-')}.epub`
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
    return { name, pages }
  }

  test.afterEach(() => {
    deleteVaultFiles(...made.splice(0))
  })

  test('toggles reflowable book table of contents listing', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // 2. Locate contents toggle button in book foot
    const showContentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(showContentsBtn).toBeVisible({ timeout: 10_000 })

    // 3. Open table of contents
    await showContentsBtn.click()

    // 4. Verify contents drawer is open with navigation
    const contentsNav = page.getByRole('group', { name: 'Contents' })
    await expect(contentsNav).toBeVisible({ timeout: 5_000 })
    await expectSearchableList(page)

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
    const { name, pages } = await openTideBook(page, request, 'chapters')

    await page.getByRole('button', { name: 'Show contents' }).click()
    const contentsNav = page.getByRole('group', { name: 'Contents' })
    const chapters = contentsNav.getByRole('option')
    await expect(chapters).toHaveText(['Dawn tide', 'Dusk tide'])
    await chapters.last().click()

    await expect(contentsNav).toBeHidden()
    await expect(pages.getByRole('heading', { name: 'Dusk tide' })).toBeVisible()

    // The plugin reads the text of the book in a task of its own. The file is deleted when the
    // test ends, and a task that reads a deleted file stays in the list of background tasks as a
    // failure and fails the tests after it. The book is found by its words once the task has written them.
    await waitForPassage(page, 'The water is low at dawn', `assets/${name}`)
    await page.keyboard.press('Escape')
    await expect(page.locator('.tasks-status__summary')).not.toHaveText(/^\d+ of \d+$/)
    await page.locator('[data-tasks-status] button').click()
    await expect(page.locator('.task-item').filter({ hasText: name })).toHaveCount(0)
    await page.keyboard.press('Escape')
  })

  test('narrows the chapters by the words typed and opens the one the keyboard is on', async ({
    page,
    request,
  }) => {
    const { name, pages } = await openTideBook(page, request, 'narrow')

    await page.getByRole('button', { name: 'Show contents' }).click()
    const contentsNav = page.getByRole('group', { name: 'Contents' })
    await contentsNav.getByRole('combobox', { name: 'Find in contents' }).fill('dusk')
    await expect(contentsNav.getByRole('option')).toHaveText(['Dusk tide'])
    await page.keyboard.press('Enter')

    await expect(contentsNav).toBeHidden()
    await expect(pages.getByRole('heading', { name: 'Dusk tide' })).toBeVisible()
    await waitForPassage(page, 'The water is low at dawn', `assets/${name}`)
  })

  test('lists the documents of a book with no table of contents by their names, and opens one', async ({
    page,
  }) => {
    await openWorkspace(page)
    await openVaultFile(page, 'assets', 'sample.epub')
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    await page.getByRole('button', { name: 'Show contents' }).click()

    const contentsNav = page.getByRole('group', { name: 'Contents' })
    const documents = contentsNav.getByRole('option')
    await expect(documents).toHaveText(['doc1'])
    await documents.first().click()
    await expect(contentsNav).toBeHidden()
  })

  test('filters book chapters using contents search field', async ({ page }) => {
    await openAsset(page, 'sample.epub')

    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Open contents panel
    const showContentsBtn = page.getByRole('button', { name: 'Show contents' })
    await expect(showContentsBtn).toBeVisible({ timeout: 10_000 })
    await showContentsBtn.click()

    const contentsNav = page.getByRole('group', { name: 'Contents' })
    await expect(contentsNav).toBeVisible({ timeout: 5_000 })

    // Locate search field
    const findField = contentsNav.getByRole('combobox', { name: 'Find in contents' })
    await expect(findField).toBeVisible()

    // Type a non-existent search query
    await findField.fill('NonExistentChapterQuery123')
    await expect(contentsNav.getByText('No table of contents.')).toBeVisible({ timeout: 5_000 })

    // Clear search
    await findField.fill('')
    await expect(findField).toHaveValue('')
  })
})
