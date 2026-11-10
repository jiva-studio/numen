import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { writeSetting } from '../../support/vault.js'
import { readFrameCalls, serveRuntime } from '../../support/runtime.js'

/**
 * The title bar the Title bar plugin draws, the window without it, and the CSS markers the Wails
 * runtime reads to move, resize and snap the window. The page is served a stand-in of the Wails
 * runtime, which records the calls of the window buttons and names the platform.
 */
const BAR_ID = 'md.numen.views.titleBar'

const getBar = (page: Page) => page.locator('[data-title-bar]')
const getControl = (page: Page, name: string) => page.locator(`[data-control="${name}"]`)
const getTopStrips = (page: Page) => page.locator('[data-workspace-strip][data-window-drag]')

test.describe('The title bar of the plugin', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [BAR_ID, 'enabled'], true)
  })

  test('draws the bar with its buttons and no strips at the top', async ({ page }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)

    await expect(getBar(page)).toBeVisible()
    await expect(getControl(page, 'close')).toBeVisible()
    await expect(getTopStrips(page)).toHaveCount(0)
  })

  test('minimizes, maximizes and closes the window through the runtime', async ({ page }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)

    await getControl(page, 'minimize').click()
    await getControl(page, 'toggle-maximize').click()
    await getControl(page, 'close').click()

    await expect
      .poll(() => readFrameCalls(page))
      .toStrictEqual(['Minimise', 'ToggleMaximise', 'Close'])
  })
})

test.describe('The window without the title bar plugin', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [BAR_ID, 'enabled'], true)
  })

  test('draws no bar and no buttons, and moves the window by the strips at the top', async ({
    page,
    request,
  }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)
    await expect(getBar(page)).toBeVisible()

    await writeSetting(request, [BAR_ID, 'enabled'], false)

    await expect(getBar(page)).toHaveCount(0)
    await expect(page.locator('[data-control]')).toHaveCount(0)
    await expect(getTopStrips(page).first()).toBeVisible()
  })

  test('draws the bar again when the plugin is enabled, without a reload', async ({
    page,
    request,
  }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)
    await writeSetting(request, [BAR_ID, 'enabled'], false)
    await expect(getBar(page)).toHaveCount(0)

    await writeSetting(request, [BAR_ID, 'enabled'], true)

    await expect(getBar(page)).toBeVisible()
    await expect(getTopStrips(page)).toHaveCount(0)
  })
})
