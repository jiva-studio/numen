import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import {
  getRow,
  getFirstLine,
  readHeight,
  showInFiles,
} from '../../support/layout.js'
import { readOwner, writeSetting } from '../../support/vault.js'

const getEditor = (page: Page, title: string) =>
  page.getByRole('tabpanel', { name: new RegExp(title) }).locator('.cm-editor')

const chooseValue = async (page: Page, value: string): Promise<void> => {
  await page.getByRole('option', { name: new RegExp(`^${value}`) }).click()
  await expect(page.getByRole('dialog', { name: 'Search the vault' })).toHaveCount(0)
}

test.describe('Window Controls and Display Preferences', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1)
  })

  test('draws note text at the reading font size chosen in the palette, also after a reload', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = getEditor(page, 'Entropy')
    await expect(editor).toBeVisible()
    const before = await readHeight(getFirstLine(editor))

    await runPaletteCommand(page, 'Reading size', 'Reading size')
    await chooseValue(page, '1.5')

    await expect.poll(async () => readHeight(getFirstLine(editor))).toBeCloseTo(before * 1.5, 0)
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ textScale: 1.5 })

    await page.reload()

    const reloaded = getEditor(page, 'Entropy')
    await expect(reloaded).toBeVisible()
    await expect.poll(async () => readHeight(getFirstLine(reloaded))).toBeCloseTo(before * 1.5, 0)
  })
})
