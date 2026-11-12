import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { createFolder, trashFiles, getRow, listPaths, showInFiles } from '../../support/layout.js'

const make = async (page: Page, command: string, name: string): Promise<void> => {
  await page.keyboard.press('Control+p')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(command)
  await expect(page.getByRole('option', { selected: true })).toContainText(command)
  await page.keyboard.press('Enter')
  await expect(field).toBeVisible()
  await field.fill(name)
  await expect(page.getByRole('option', { selected: true })).toContainText(name)
  await page.keyboard.press('Enter')
}

test.describe('a deck or a card model made through the palette', () => {
  test('a deck opens in the editor of its cards, at the path the vault filed it under', async ({
    page,
  }) => {
    await openWorkspace(page)
    await make(page, 'Create deck', 'Rivers')

    await expect(page.getByRole('tab', { name: /Rivers/ }).first()).toBeVisible()
    await expect(page.locator('.deck').first()).toBeVisible()
    await expect(page.locator('[data-tree-row="Rivers.md"]')).toHaveCount(1)
  })

})

test.describe('a deck made from the New submenu of a row of the file tree', () => {
  test('is filed in the folder of the row under the name typed', async ({ page, request }) => {
    await createFolder(request, 'Glaciers')
    try {
      await openWorkspace(page)
      await showInFiles(page, [])
      await getRow(page, 'Glaciers').click({ button: 'right' })
      const newItem = page.getByRole('menuitem', { name: 'New', exact: true })
      await newItem.focus()
      await newItem.press('ArrowRight')
      await page.getByRole('menuitem', { name: 'Create deck', exact: true }).click()
      const field = page.locator('input[data-palette="field"]').first()
      await expect(field).toBeVisible()
      await field.fill('Moraines')
      await expect(page.getByRole('option', { selected: true })).toContainText('Moraines')
      await page.keyboard.press('Enter')

      await expect
        .poll(async () => listPaths(request, 'Glaciers'))
        .toEqual(['Glaciers/Moraines.md'])
    } finally {
      await trashFiles(request, ['Glaciers'])
    }
  })
})

test.describe('a file open in an editor of cards, removed from the tree', () => {
  for (const [what, path, editor] of [
    ['deck', 'cards/empty-deck.md', '.deck'],
    ['card-model', 'cards/fieldless-card-model.md', '.card-model-tab'],
  ] as const) {
    test(`releases the tab holding a ${what}`, async ({ page }) => {
      await openWorkspace(page)
      await page.getByRole('tab', { name: /^Files/ }).click()
      const folder = page.locator('[data-tree-row="cards"]')
      await folder.press('Enter')
      const row = page.locator(`[data-tree-row="${path}"]`)
      await row.press('Enter')
      await expect(page.locator(editor).first()).toBeVisible()

      await page.getByRole('tab', { name: /^Files/ }).click()
      await row.click({ button: 'right' })
      await page.getByRole('menuitem', { name: 'Move to trash', exact: true }).first().click()

      await expect(page.locator(editor)).toHaveCount(0)
    })
  }
})
