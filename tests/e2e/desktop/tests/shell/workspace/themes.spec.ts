import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand, runPaletteCommand } from '../../support/workspace.js'
import { getRow, readComputedFontSize, showInFiles } from '../../support/layout.js'
import { readSettings, writeSetting } from '../../support/vault.js'

/** The colour the window is painted in by the default theme, in each half. */
const SURFACE = { light: 'rgb(250, 244, 232)', dark: 'rgb(30, 25, 22)' } as const

/** The colour `--numen-surface` resolves to on the page. */
const readSurface = (page: Page): Promise<string> =>
  page.evaluate(() => {
    const probe = document.createElement('div')
    probe.style.color = 'var(--numen-surface)'
    document.body.append(probe)
    const colour = getComputedStyle(probe).color
    probe.remove()
    return colour
  })

/** The size of the text the interface is drawn in, which every rem of it is measured in. */
const readRootFontSize = (page: Page): Promise<number> =>
  page.evaluate(() => Number.parseFloat(getComputedStyle(document.documentElement).fontSize))

const setMode = async (page: Page, name: 'Light' | 'Dark'): Promise<void> => {
  await runPaletteCommand(page, 'Light or dark', 'Light or dark')
  await page.getByRole('option', { name: new RegExp(`^${name}`) }).click()
}

/** Opens the system section of the settings tab, and gives back the tab. */
const openSystemSettings = async (page: Page) => {
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /^System/ }).click()
  return settings
}

test.describe('Theming and Display Scaling', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['appearance', 'mode'], 'system')
    await writeSetting(request, ['appearance', 'interface_scale'], 1)
    await writeSetting(request, ['appearance', 'text_scale'], 1)
  })

  test('paints the window in the dark half of the theme once Dark is chosen', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await setMode(page, 'Dark')

    await expect.poll(() => readSurface(page)).toBe(SURFACE.dark)
    await expect
      .poll(async () => (await readSettings(request))['appearance'])
      .toMatchObject({ mode: 'dark' })
  })

  test('paints the window in the light half of the theme once Light is chosen', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await setMode(page, 'Light')

    await expect.poll(() => readSurface(page)).toBe(SURFACE.light)
    await expect
      .poll(async () => (await readSettings(request))['appearance'])
      .toMatchObject({ mode: 'light' })
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
      .poll(async () => (await readSettings(request))['appearance'])
      .toMatchObject({ interface_scale: 1.2, text_scale: 1.5 })

    await page.reload()

    await expect.poll(() => readRootFontSize(page)).toBeCloseTo(rootBefore * 1.2, 0)
    const reloaded = page.getByRole('tabpanel', { name: /Entropy/ }).locator('.cm-editor')
    await page.getByRole('tab', { name: /^Entropy/ }).click()
    await expect(reloaded).toBeVisible()
    await expect
      .poll(async () => readComputedFontSize(reloaded))
      .toBeCloseTo(editorBefore * 1.2 * 1.5, 0)
  })
})
