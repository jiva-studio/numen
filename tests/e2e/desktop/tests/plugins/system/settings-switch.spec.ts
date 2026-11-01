import { test, expect, type Locator, type Page } from '../../support/test.js'
import { openWorkspace, runCommand } from '../../support/workspace.js'

const PLUGIN = 'ACP Agents'
const PLUGIN_ID = 'md.numen.agents.acp'

declare global {
  interface Window {
    /** Every `aria-checked` the observed switch has shown, in order. */
    shownChecked?: string[]
  }
}

const openPluginPage = async (page: Page, name: string): Promise<Locator> => {
  await runCommand(page, 'Settings', /^Settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: new RegExp(name) }).click()
  return settings
}

/**
 * Records every value of `aria-checked` the switch named `label` shows from now on, starting with
 * the current one, and `none` while the page draws no such switch.
 */
const recordChecked = async (page: Page, label: string): Promise<void> => {
  await page.evaluate((name) => {
    const read = () =>
      document
        .querySelector(`[role="switch"][aria-label="${name}"]`)
        ?.getAttribute('aria-checked') ?? 'none'
    window.shownChecked = [read()]
    new MutationObserver(() => {
      const value = read()
      if (window.shownChecked?.at(-1) !== value) window.shownChecked?.push(value)
    }).observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ['aria-checked'],
    })
  }, label)
}

/** Whether the settings file holds the plugin as enabled, as the core reads it back. */
const readEnabled = async (page: Page): Promise<boolean> => {
  const answer = await page.request.post('/numen.v1.SettingsService/GetSettings', {
    data: {},
    headers: { 'Content-Type': 'application/json' },
  })
  const settingsFile = JSON.parse((await answer.json()).written) as {
    plugins?: Record<string, { enabled?: boolean }>
  }
  return settingsFile.plugins?.[PLUGIN_ID]?.enabled !== false
}

const toggleAndRecord = async (page: Page, toggle: Locator, isEnabled: boolean) => {
  await recordChecked(page, `Enabled ${PLUGIN}`)
  await toggle.click()
  await expect.poll(() => readEnabled(page)).toBe(isEnabled)
  // The window reads the settings again after a write, the last time 16 s after it.
  await page.waitForTimeout(17_000)
  return page.evaluate(() => window.shownChecked ?? [])
}

test.describe('The enabled switch of a plugin', () => {
  test.describe.configure({ timeout: 120_000 })

  test.afterEach(async ({ page }) => {
    await page.request.post('/numen.v1.SettingsService/WriteSettings', {
      data: { settings: [{ at: ['plugins', PLUGIN_ID, 'enabled'], value: 'false' }] },
      headers: { 'Content-Type': 'application/json' },
    })
  })

  test('shows the old value and then the new one, once each, on enable and disable', async ({
    page,
  }) => {
    await openWorkspace(page)
    const settings = await openPluginPage(page, PLUGIN)
    const toggle = settings.getByRole('switch', { name: new RegExp(`Enabled ${PLUGIN}`) })
    await expect(toggle).toHaveAttribute('aria-checked', 'false')

    expect(await toggleAndRecord(page, toggle, true)).toEqual(['false', 'true'])
    expect(await toggleAndRecord(page, toggle, false)).toEqual(['true', 'false'])
  })
})
