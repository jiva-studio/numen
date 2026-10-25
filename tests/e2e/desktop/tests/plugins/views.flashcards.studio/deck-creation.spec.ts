import { test, expect, type Page } from 'playwright/test'
import { openWorkspace } from '../../support/workspace'

/** A deck or a stencil made through the palette, under the name typed on its step. */
const make = async (page: Page, command: string, name: string): Promise<void> => {
  await page.keyboard.press('Control+p')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(command)
  await page.keyboard.press('Enter')
  await expect(field).toBeVisible()
  await field.fill(name)
  await page.keyboard.press('Enter')
}

test.describe('a deck or a stencil made through the palette', () => {
  test('a deck opens in the editor of its cards, at the path the vault filed it under', async ({
    page,
  }) => {
    await openWorkspace(page)
    await make(page, 'New deck', 'Rivers')

    await expect(page.getByRole('tab', { name: /Rivers/ }).first()).toBeVisible()
    await expect(page.locator('.deck').first()).toBeVisible()
    await expect(page.locator('[data-tree-row="Rivers.md"]')).toHaveCount(1)
  })

  test('reports a taken name and opens no second tab', async ({ page }) => {
    await openWorkspace(page)
    await make(page, 'New deck', 'Lakes')
    await expect(page.locator('.deck').first()).toBeVisible()

    await make(page, 'New deck', 'Lakes')

    await expect(page.getByText(/filed there already/).first()).toBeVisible()
    await expect(page.getByRole('tab', { name: /Lakes/ })).toHaveCount(1)
  })

  test('a stencil opens in the editor of its fields and faces, and no deck opens', async ({
    page,
  }) => {
    await openWorkspace(page)
    await make(page, 'New stencil', 'Mountain')

    await expect(page.getByRole('tab', { name: /Mountain/ }).first()).toBeVisible()
    await expect(page.locator('.stencil-tab').first()).toBeVisible()
    await expect(page.locator('.deck')).toHaveCount(0)
  })
})

test.describe('a file open in an editor of cards, removed from the tree', () => {
  for (const [what, path, editor] of [
    ['deck', 'cards/empty-deck.md', '.deck'],
    ['stencil', 'cards/fieldless-stencil.md', '.stencil-tab'],
  ] as const) {
    test(`lets go of the tab holding a ${what}`, async ({ page }) => {
      await openWorkspace(page)
      await page.getByText('Files').first().click()
      const folder = page.locator('[data-tree-row="cards"]')
      await folder.press('Enter')
      const row = page.locator(`[data-tree-row="${path}"]`)
      await row.press('Enter')
      await expect(page.locator(editor).first()).toBeVisible()

      await page.getByText('Files').first().click()
      await row.click({ button: 'right' })
      await page
        .getByRole('menuitem', { name: /Move to trash|Remove/ })
        .first()
        .click()

      await expect(page.locator(editor)).toHaveCount(0)
    })
  }
})
