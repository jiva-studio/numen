import { test, expect, type Page } from 'playwright/test'
import { openWorkspace } from '../../support/workspace'

/**
 * A path opens in the editor its file is for, whichever road it was reached by: a deck in the
 * editor of its cards, a stencil in the editor of its fields and faces, a note in the editor of its
 * prose, a book in a reader.
 */

/** The row of a file in the tree, its folder opened first. */
const openFromTree = async (page: Page, folder: string, path: string): Promise<void> => {
  await page.getByRole('tab', { name: /^Files/ }).click()
  const folderRow = page.locator(`[data-tree-row="${folder}"]`)
  await expect(folderRow).toBeVisible()
  await folderRow.press('Enter')
  const row = page.locator(`[data-tree-row="${path}"]`)
  await expect(row).toBeVisible()
  await row.press('Enter')
}

/**
 * The file the search of the vault turned up under its name, opened as a file: Enter takes the
 * row's first action, which is the plex, and Shift with Enter its second.
 */
const openFromSearch = async (page: Page, name: string): Promise<void> => {
  await page.keyboard.press('Control+k')
  const field = page.locator('input[placeholder="Search the vault"]')
  await expect(field).toBeVisible()
  await field.fill(name)
  const row = page
    .locator('[data-palette="list"] [role="option"]')
    .filter({ has: page.locator('[data-palette="name"]', { hasText: new RegExp(`^${name}$`) }) })
    .first()
  await row.hover()
  await page.keyboard.press('Shift+Enter')
}

const ROADS = {
  'the tree': openFromTree,
  'the search': (page: Page, _folder: string, path: string) =>
    openFromSearch(page, path.split('/').pop()!.replace(/\.md$/, '')),
}

for (const [road, open] of Object.entries(ROADS)) {
  test.describe(`a file reached by ${road}`, () => {
    test('opens a deck in the editor of its cards', async ({ page }) => {
      await openWorkspace(page)
      await open(page, 'cards', 'cards/Animals.md')

      await expect(page.locator('.deck').first()).toBeVisible()
      await expect(page.locator('.stencil-tab')).toHaveCount(0)
    })

    test('opens a stencil in the editor of its fields and faces', async ({ page }) => {
      await openWorkspace(page)
      await open(page, 'cards', 'cards/Animal.md')

      await expect(page.locator('.stencil-tab').first()).toBeVisible()
      await expect(page.locator('.deck')).toHaveCount(0)
    })

    test('opens an ordinary note in the editor of its prose', async ({ page }) => {
      await openWorkspace(page)
      await open(page, 'notes', 'notes/Entropy.md')

      await expect(page.locator('.note').first()).toBeVisible()
      await expect(page.locator('.deck')).toHaveCount(0)
      await expect(page.locator('.stencil-tab')).toHaveCount(0)
    })
  })
}

test('a book reached by the tree opens in a reader and in no note editor', async ({ page }) => {
  await openWorkspace(page)
  await openFromTree(page, 'assets', 'assets/sample.pdf')

  await expect(page.locator('.reader').first()).toBeVisible()
  await expect(page.locator('.note')).toHaveCount(0)
})
