import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'
import { getRow, readComputedFontSize, showInFiles } from '../../support/layout.js'
import { readOwner, writeSetting } from '../../support/vault.js'

/** The size of the text the interface is drawn in, which every rem of it is measured in. */
const readRootFontSize = (page: Page): Promise<number> =>
  page.evaluate(() => Number.parseFloat(getComputedStyle(document.documentElement).fontSize))

/** Opens the system section of the settings tab, and gives back the tab. */
const openSystemSettings = async (page: Page) => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /^System/ }).click()
  return settings
}

test.describe('Theming and Display Scaling', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'appearance', 'mode'], 'system')
    await writeSetting(request, ['app', 'appearance', 'interfaceScale'], 1)
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1)
  })

  test('draws the interface and the note text at the sizes typed in the settings tab, also after a reload', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, 'notes/Entropy.md').press('Enter')
    const editor = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await expect(editor).toBeVisible()
    const rootBefore = await readRootFontSize(page)
    const editorBefore = await readComputedFontSize(editor)

    const settings = await openSystemSettings(page)
    const interfaceSize = settings.getByRole('spinbutton', { name: 'Interface size' })
    await interfaceSize.fill('1.2')
    await interfaceSize.press('Enter')
    await expect.poll(() => readRootFontSize(page)).toBeCloseTo(rootBefore * 1.2, 0)

    const readingSize = settings.getByRole('spinbutton', { name: 'Reading size' })
    await readingSize.fill('1.5')
    await readingSize.press('Enter')
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ interfaceScale: 1.2, textScale: 1.5 })

    await page.reload()

    await expect.poll(() => readRootFontSize(page)).toBeCloseTo(rootBefore * 1.2, 0)
    const reloaded = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await page.getByRole('tab', { name: /^Entropy/ }).click()
    await expect(reloaded).toBeVisible()
    await expect
      .poll(async () => readComputedFontSize(reloaded))
      .toBeCloseTo(editorBefore * 1.2 * 1.5, 0)
  })

  test('refuses a size outside the bounds of its setting and keeps the size in force', async ({
    request,
  }) => {
    await writeSetting(request, ['app', 'appearance', 'textScale'], 1.25)

    const refused = await request.post('/numen.v1.SettingsService/WriteSettings', {
      data: { settings: [{ at: ['app', 'appearance', 'textScale'], value: '9' }] },
      headers: { 'Content-Type': 'application/json' },
    })

    expect(refused.ok()).toBe(false)
    expect((await readOwner(request, 'app'))['appearance']).toMatchObject({ textScale: 1.25 })
  })
})
