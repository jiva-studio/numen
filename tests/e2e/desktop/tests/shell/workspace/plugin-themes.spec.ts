import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

/** `--numen-surface` in the Dracula theme of the preset themes plugin. */
const DRACULA_SURFACE = '#21222c'

const getThemeStyle = (page: Page): Promise<string> =>
  page.evaluate(
    () => document.head.querySelector('style[data-appearance="theme"]')?.textContent ?? '',
  )

const getServedThemeStyle = async (page: Page): Promise<string> => {
  const html = await (await page.request.get('/')).text()
  return /<style data-appearance="theme">([^<]*)<\/style>/.exec(html)?.[1] ?? ''
}

const reloadWindow = async (page: Page): Promise<void> => {
  await page.reload()
  await expect(page.getByRole('tabpanel').first()).toBeVisible()
}

const chooseTheme = async (page: Page, title: string): Promise<void> => {
  await runCommand(page, 'Theme', /^Theme/)
  await page
    .getByRole('option', { name: new RegExp(`^${title}`) })
    .first()
    .click()
}

const setPresetThemesEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /^(Linked tab )?Settings Close Settings$/ })
  await expect(settings).toBeVisible()
  const toggle = settings.getByRole('switch', { name: /Enabled Preset Themes/ })
  if (!(await toggle.isVisible())) {
    await settings.getByRole('button', { name: /Preset Themes/ }).click()
  }
  // The switch moves when the core has answered its write, so the click is not waited for to change it.
  if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('A theme of the preset themes plugin', () => {
  test.describe.configure({ timeout: 120_000 })

  test('is applied at once, is in the first paint after a reload, and gives way to the default when the plugin is disabled', async ({
    page,
  }) => {
    await openWorkspace(page)
    try {
      await chooseTheme(page, 'Dracula')
      await expect.poll(() => getThemeStyle(page)).toContain(DRACULA_SURFACE)

      // The server writes the theme into the page it sends, so the first frame is already drawn in it.
      await expect.poll(() => getServedThemeStyle(page)).toContain(DRACULA_SURFACE)
      await reloadWindow(page)
      await expect.poll(() => getThemeStyle(page)).toContain(DRACULA_SURFACE)

      await setPresetThemesEnabled(page, false)
      await expect.poll(() => getServedThemeStyle(page)).not.toContain(DRACULA_SURFACE)
      await reloadWindow(page)
      await expect.poll(() => getThemeStyle(page)).not.toContain(DRACULA_SURFACE)
    } finally {
      await setPresetThemesEnabled(page, true)
      await chooseTheme(page, 'numen')
    }
  })
})
