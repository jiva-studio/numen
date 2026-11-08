import { test, expect, type Page } from '../../support/test.js'
import { openVaultFile, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle, readNote } from '../../support/vault.js'

const createCardModel = async (page: Page, title: string): Promise<void> => {
  await runPaletteCommand(page, 'Create card model', 'Create card model')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(title)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('tab', { name: new RegExp(title) })).toBeVisible()
}

test.describe('the card model editor', () => {
  let title: string

  test.beforeEach(() => {
    title = createUniqueTitle('Rock pool')
  })

  test.afterEach(() => {
    deleteVaultFiles(`${title}.md`)
  })

  test('lists the fields the card model file declares, in its order', async ({ page }) => {
    await openWorkspace(page)
    const panel = await openVaultFile(page, 'cards', 'Term.md')

    const fields = panel.getByRole('region', { name: 'Fields' })
    await expect(fields.getByRole('textbox')).toHaveCount(4)
    await expect(fields.getByRole('textbox', { name: 'Field 1' })).toHaveValue('Слово')
    await expect(fields.getByRole('textbox', { name: 'Field 4' })).toHaveValue(
      'Имя поля с пробелом',
    )
  })

  test('writes an added template and the front typed into it to the card model file', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await createCardModel(page, title)
    const templates = page.getByRole('region', { name: 'Templates' })

    await templates.getByRole('button', { name: 'Add a template' }).click()
    const front = templates.locator('textarea[data-half="front"]').first()
    await front.fill('What lives under {{Field 1}}?')

    await expect
      .poll(async () => (await readNote(request, `${title}.md`)).body)
      .toMatch(/^## .+\n\n### Front\n\nWhat lives under \{\{Field 1\}\}\?$/m)
  })
})
