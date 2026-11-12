import { test, expect } from '../../support/test.js'
import { openVaultFile, openWorkspace } from '../../support/workspace.js'

test.describe('EPUB Book Reader Contents and Chapter Navigation', () => {
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
})
