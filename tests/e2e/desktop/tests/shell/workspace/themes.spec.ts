import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'
import { getFirstLine, getRow, readHeight, showInFiles } from '../../support/layout.js'
import { readOwner, writeSetting } from '../../support/vault.js'

const openSystemSettings = async (page: Page) => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /^(Linked tab )?Settings Close Settings$/ })
  await settings.getByRole('button', { name: /^System/ }).click()
  return settings
}

test.describe('Theming and Display Scaling', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'appearance', 'mode'], 'system')
    await writeSetting(request, ['app', 'appearance', 'interfaceScale'], 1)
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1)
  })

  test('sets the interface size and the reading size in the settings tab, and keeps them after a reload', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await expect(editor).toBeVisible()
    const editorBefore = await readHeight(getFirstLine(editor))

    const settings = await openSystemSettings(page)
    const interfaceSize = settings.getByRole('slider', { name: 'Interface size' })
    await interfaceSize.focus()
    await interfaceSize.press('ArrowRight')
    await interfaceSize.press('ArrowRight')

    const readingSize = settings.getByRole('slider', { name: 'Reading size' })
    await readingSize.focus()
    for (let step = 0; step < 5; step++) await readingSize.press('ArrowRight')
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ interfaceScale: 1.2, textScale: 1.5 })

    await page.reload()

    const reloaded = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await page.getByRole('tab', { name: /^Entropy/ }).click()
    await expect(reloaded).toBeVisible()
    await expect
      .poll(async () => readHeight(getFirstLine(reloaded)))
      .toBeCloseTo(editorBefore * 1.2 * 1.5, 0)
  })
})
