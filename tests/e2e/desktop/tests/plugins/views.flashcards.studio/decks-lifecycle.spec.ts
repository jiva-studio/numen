import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle, readNote, readVaultFile } from '../../support/vault.js'

/** Runs `Create deck` from the palette under the name given on its step. */
const createDeck = async (page: Page, title: string): Promise<void> => {
  await runPaletteCommand(page, 'Create deck', 'Create deck')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(title)
  await page.keyboard.press('Enter')
}

test.describe('a deck made through the palette', () => {
  let title: string

  test.beforeEach(() => {
    title = createUniqueTitle('Puffin colony')
  })

  test.afterEach(() => {
    deleteVaultFiles(`${title}.md`)
  })

  test('is a deck file named after it at the root of the vault, open in the deck editor', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await createDeck(page, title)

    await expect(page.getByRole('tab', { name: new RegExp(title) })).toBeVisible()
    await expect(page.locator('.deck').first()).toBeVisible()
    await expect(page.locator('.card')).toHaveCount(0)
    const text = await readVaultFile(request, `${title}.md`)
    expect(text).toMatch(/^type: deck$/m)
  })

  test('writes its first card under the card model chosen for it', async ({ page, request }) => {
    await openWorkspace(page)
    await createDeck(page, title)
    await expect(page.locator('.deck').first()).toBeVisible()

    await page.getByRole('button', { name: 'Add a card' }).first().click()
    await page.locator('[data-cut="Animal"]').click()
    await page
      .getByRole('article', { name: 'Card 1' })
      .getByRole('textbox', { name: 'Name', exact: true })
      .fill('Puffin')

    await expect
      .poll(async () => (await readNote(request, `${title}.md`)).body)
      .toContain('### Name\n\nPuffin')
    const body = (await readNote(request, `${title}.md`)).body
    expect(body).toMatch(/^## Puffin \^[0-9a-hjkmnp-tv-z]{10}\n\n\[\[Animal\]\]$/m)
  })
})
