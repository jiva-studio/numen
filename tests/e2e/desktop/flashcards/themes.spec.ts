import { expect, test, type Page } from '../tests/support/certificate.js'

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

const chooseTheme = async (page: Page, title: string): Promise<void> => {
  await page.keyboard.press('Control+p')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(title)
  await page
    .getByRole('option', { name: new RegExp(`^${title}`) })
    .first()
    .click()
}

test('a theme of the preset themes plugin is chosen in the command palette and kept across a reload', async ({
  page,
}) => {
  await page.goto('/')
  try {
    await chooseTheme(page, 'Dracula')
    await expect.poll(() => getThemeStyle(page)).toContain(DRACULA_SURFACE)

    await expect.poll(() => getServedThemeStyle(page)).toContain(DRACULA_SURFACE)
    await page.reload()
    await expect.poll(() => getThemeStyle(page)).toContain(DRACULA_SURFACE)
  } finally {
    await chooseTheme(page, 'numen')
  }
  await expect.poll(() => getThemeStyle(page)).not.toContain(DRACULA_SURFACE)
})
