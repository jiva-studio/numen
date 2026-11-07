import { test, expect } from '../../support/test.js'
import { writePluginSetting } from '../../support/media.js'
import { readOwner } from '../../support/vault.js'
import { waitForWindow } from '../../support/workspace.js'

const STATUS_BAR = 'md.numen.views.statusBar'

test.describe('The status bar plugin', () => {
  test.afterEach(async ({ request }) => {
    await writePluginSetting(request, STATUS_BAR, 'enabled', false)
  })

  test('is off by default, and the window has no bar', async ({ page, request }) => {
    expect((await readOwner(request, STATUS_BAR))['enabled']).toBe(false)

    await page.goto('/')
    await waitForWindow(page)

    await expect(page.getByRole('region', { name: 'Status bar' })).toHaveCount(0)
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
