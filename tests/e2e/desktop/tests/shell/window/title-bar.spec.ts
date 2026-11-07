import { test, expect, type Locator, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import { writeSetting } from '../../support/vault.js'
import { usePlatformAgent } from '../../support/browser.js'

/**
 * The title bar the Title bar plugin draws, the window without it, and the CSS markers the Wails
 * runtime reads to move, resize and snap the window. The page is served a stand-in of the Wails
 * runtime, which records the calls of the window buttons and names the platform.
 */
const BAR_ID = 'md.numen.views.titleBar'

type Platform = 'windows' | 'mac'

/** A module in the place of /wails/runtime.js: the window state, and a list of the calls made. */
const createRuntime = (platform: Platform): string => `
const calls = (globalThis.__frameCalls ??= [])
export const Window = {
  Minimise: async () => { calls.push('Minimise') },
  ToggleMaximise: async () => { calls.push('ToggleMaximise') },
  Close: async () => { calls.push('Close') },
  IsMaximised: async () => false,
  IsFullscreen: async () => false,
}
export const System = {
  IsMac: () => ${platform === 'mac'},
  IsWindows: () => ${platform === 'windows'},
  IsLinux: () => false,
}
`

/** Serves the runtime of the platform, and names the platform in the user agent the plugins read. */
const serveRuntime = async (page: Page, platform: Platform): Promise<void> => {
  await usePlatformAgent(page, platform)
  await page.route('**/wails/runtime.js', (route) =>
    route.fulfill({ contentType: 'text/javascript', body: createRuntime(platform) }),
  )
}

const readFrameCalls = (page: Page): Promise<string[]> =>
  page.evaluate(() => [...((globalThis as { __frameCalls?: string[] }).__frameCalls ?? [])])

/** The value of a custom property as the element computes it, inherited ones included. */
const readProperty = (locator: Locator, name: string): Promise<string> =>
  locator.evaluate(
    (element, property) => getComputedStyle(element).getPropertyValue(property).trim(),
    name,
  )

const getBar = (page: Page) => page.locator('[data-title-bar]')
const getControl = (page: Page, name: string) => page.locator(`[data-control="${name}"]`)
const getTopStrips = (page: Page) => page.locator('[data-workspace-strip][data-window-drag]')

test.describe('The title bar of the plugin', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [BAR_ID, 'enabled'], true)
  })

  test('moves the window by the bar and not by its buttons', async ({ page }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)

    await expect(getBar(page)).toBeVisible()
    expect(await readProperty(getBar(page), '--wails-draggable')).toBe('drag')
    expect(
      await readProperty(page.locator('[data-title-bar-command]').first(), '--wails-draggable'),
    ).toBe('no-drag')
    expect(await readProperty(getControl(page, 'close'), '--wails-draggable')).toBe('no-drag')
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

  test('marks the window buttons for Snap Layouts and the page for resizing by its edges', async ({
    page,
  }) => {
    await serveRuntime(page, 'windows')
    await openWorkspace(page)

    expect(
      await readProperty(getControl(page, 'toggle-maximize'), '--wails-non-client-region'),
    ).toBe('maximize')
    expect(await readProperty(getControl(page, 'minimize'), '--wails-non-client-region')).toBe(
      'minimize',
    )
    expect(await readProperty(getControl(page, 'close'), '--wails-non-client-region')).toBe('close')
    expect(await readProperty(page.locator('body'), '--wails-resize')).toBe('all')
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
    expect(await readProperty(getTopStrips(page).first(), '--wails-draggable')).toBe('drag')
    expect(await readProperty(page.getByRole('tab').first(), '--wails-draggable')).toBe('no-drag')
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

  test('keeps room for the macOS buttons in the strip at the top-left corner', async ({
    page,
    request,
  }) => {
    await serveRuntime(page, 'mac')
    await writeSetting(request, [BAR_ID, 'enabled'], false)
    await openWorkspace(page)

    const first = getTopStrips(page).first()
    await expect(first).toBeVisible()
    await expect
      .poll(() => first.evaluate((element) => getComputedStyle(element).paddingLeft))
      .toBe('80px')
  })
})
