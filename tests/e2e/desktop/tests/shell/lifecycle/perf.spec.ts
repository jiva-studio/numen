import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace, runCommand, runPaletteCommand } from '../../support/workspace.js'
import { readOwner, writeSetting } from '../../support/vault.js'

/** The profile the palette opens its list on, which is the one in force, by the name of its row. */
const readCurrentProfile = async (page: Page): Promise<string> => {
  await runPaletteCommand(page, 'Set performance profile', 'Set performance profile')
  const current = page.getByRole('option', { selected: true })
  await expect(current).toHaveCount(1)
  const name = (await current.locator('[data-palette="name"]').textContent()) ?? ''
  await page.keyboard.press('Escape')
  return name.trim()
}

test.describe('App Updates, Performance Profiles and OCR Settings', () => {
  test.describe.configure({ timeout: 90_000 })

  let profile: unknown
  let mode: unknown
  /** The row of the profile other than the one in force. */
  let other = ''

  test.beforeEach(async ({ request }) => {
    const app = await readOwner(request, 'app')
    profile = (app['performance'] as Record<string, unknown> | undefined)?.['profile']
    mode = (app['appearance'] as Record<string, unknown> | undefined)?.['mode']
    other = profile === 'eco' ? 'Maximum' : 'Eco'
  })

  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['app', 'performance', 'profile'], profile)
    await writeSetting(request, ['app', 'appearance', 'mode'], mode)
  })

  test('keeps the performance profile chosen in the palette in the settings, also after a reload', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)

    await runPaletteCommand(page, 'Set performance profile', 'Set performance profile')
    await page.getByRole('option', { name: new RegExp(`^${other}`) }).click()

    await expect
      .poll(async () => (await readOwner(request, 'app'))['performance'])
      .toMatchObject({ profile: other.toLowerCase() })

    await page.reload()
    expect(await readCurrentProfile(page)).toBe(other)
  })

  test('inspects ocr engine language options in settings tab', async ({ page }) => {
    await openWorkspace(page)
    await runCommand(page, 'Open settings', /^Open settings$/)
    const settings = page.getByRole('tabpanel', { name: /Settings/ })

    await settings
      .getByRole('button', { name: /Document OCR/i })
      .first()
      .click()

    await expect(settings.locator('.plugin-detail__title')).toContainText(/OCR/)
  })

  test('persists the dark mode across a full application reload', async ({ page, request }) => {
    await openWorkspace(page)
    await runPaletteCommand(page, 'Choose light or dark mode', 'Choose light or dark mode')
    await page.getByRole('option', { name: /^Dark/ }).click()

    await expect
      .poll(() => page.evaluate(() => getComputedStyle(document.documentElement).colorScheme))
      .toContain('dark')
    await expect
      .poll(async () => (await readOwner(request, 'app'))['appearance'])
      .toMatchObject({ mode: 'dark' })

    await page.reload()
    await expect(page.getByRole('group', { name: 'Neighbourhood' })).toBeVisible()

    expect(
      await page.evaluate(() => getComputedStyle(document.documentElement).colorScheme),
    ).toContain('dark')
  })
})
