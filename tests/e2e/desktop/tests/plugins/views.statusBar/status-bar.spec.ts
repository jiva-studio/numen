import fs from 'node:fs'
import type { APIRequestContext } from 'playwright/test'
import { test, expect } from '../../support/test.js'
import { writePluginSetting } from '../../support/media.js'
import { callRPC } from '../../support/vault.js'
import { waitForWindow } from '../../support/workspace.js'

const STATUS_BAR = 'md.numen.views.statusBar'

/** The plugins an installation runs only once the person turns them on, as the core ships them. */
const DEFAULT_OFF = (
  JSON.parse(
    fs.readFileSync(
      new URL('../../../../../../libs/core/kernel/adapter/settings/plugins.json', import.meta.url),
      'utf8',
    ),
  ) as { off: readonly string[] }
).off

const isPluginActive = async (request: APIRequestContext, pluginId: string): Promise<boolean> => {
  const answer = await callRPC(request, 'SettingsService/GetSettings', {})
  const plugins = (answer['plugins'] ?? []) as readonly { id?: string; isActive?: boolean }[]
  return plugins.find((one) => one.id === pluginId)?.isActive === true
}

test.describe('The status bar plugin', () => {
  test.afterEach(async ({ request }) => {
    await writePluginSetting(request, STATUS_BAR, 'enabled', false)
  })

  test('is off by default, and the window has no bar', async ({ page, request }) => {
    expect(DEFAULT_OFF).toContain(STATUS_BAR)

    await page.goto('/')
    await waitForWindow(page)

    await expect.poll(() => isPluginActive(request, STATUS_BAR)).toBe(false)
    await expect(page.getByRole('region', { name: 'Status bar' })).not.toBeAttached()
  })

  test('draws the bar at the bottom of the window once it is turned on', async ({
    page,
    request,
  }) => {
    await writePluginSetting(request, STATUS_BAR, 'enabled', true)

    await page.goto('/')
    await waitForWindow(page)

    const bar = page.getByRole('region', { name: 'Status bar' })
    await expect(bar).toBeVisible()
    const box = await bar.boundingBox()
    expect((box?.y ?? 0) + (box?.height ?? 0)).toBeCloseTo(page.viewportSize()?.height ?? 0, 0)
  })
})
