import { test, expect, type Page } from '../../support/test.js'
import { openVaultFile, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import { deleteVaultFiles } from '../../support/media.js'
import { createUniqueTitle, readNote, readVaultFile } from '../../support/vault.js'

/** Runs `New card model` from the palette under the name given on its step. */
const createCardModel = async (page: Page, title: string): Promise<void> => {
  await runPaletteCommand(page, 'New card model', 'New card model')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(title)
  await page.keyboard.press('Enter')
  await expect(page.getByRole('tab', { name: new RegExp(title) })).toBeVisible()
}

/** The names under `fields:` in the frontmatter of a card model file. */
const readFields = (text: string): string[] =>
  [...(/^fields:\n((?: {2}- .*\n)*)/m.exec(text)?.[1]?.matchAll(/^ {2}- (.*)$/gm) ?? [])].map(
    (one) => one[1] ?? '',
  )

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

  test('New card model writes a card model file holding its first field', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await createCardModel(page, title)

    await expect(
      page.getByRole('region', { name: 'Fields' }).getByRole('textbox', { name: 'Field 1' }),
    ).toHaveValue('Field 1')
    const text = await readVaultFile(request, `${title}.md`)
    expect(text).toMatch(/^type: card-model$/m)
    expect(readFields(text)).toEqual(['Field 1'])
  })

  test('writes an added field to the frontmatter of the card model file', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await createCardModel(page, title)
    const fields = page.getByRole('region', { name: 'Fields' })

    await fields.getByRole('button', { name: 'Add a field' }).click()
    const added = fields.getByRole('textbox', { name: 'Field 2' })
    await added.fill('Depth')
    await added.press('Enter')

    await expect
      .poll(async () => readFields(await readVaultFile(request, `${title}.md`)))
      .toEqual(['Field 1', 'Depth'])

    await page.reload()
    await openWorkspace(page)
    await page.getByRole('tab', { name: /^Files/ }).click()
    await page.getByRole('treeitem', { name: `${title}.md`, exact: true }).press('Enter')
    await expect(
      page.getByRole('region', { name: 'Fields' }).getByRole('textbox', { name: 'Field 2' }),
    ).toHaveValue('Depth')
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
