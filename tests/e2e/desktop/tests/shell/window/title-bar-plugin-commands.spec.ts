import { test, expect, type Page } from '../../support/test.js'
import { closeTab, getPaletteField, openWorkspace } from '../../support/workspace.js'
import { getTab } from '../../support/layout.js'
import { writeSetting } from '../../support/vault.js'

/**
 * The buttons the plugins of the application place in `titleBar`. They need a plugin that places
 * a command there; the file tree, the plex graph and the editor command Create note are the
 * entries the plugins of the application hold.
 */
const TREE_ID = 'md.numen.views.file.tree'
const TREE_BUTTON = `${TREE_ID}:tree.open`
const GRAPH_BUTTON = 'md.numen.views.plex.graph:graph.open'
const NOTE_BUTTON = 'app:note.create'
const SETTINGS_BUTTON = 'app:settings.open'

const getButton = (page: Page, id: string) => page.locator(`[data-title-bar-command="${id}"]`)
const getButtons = (page: Page) => page.locator('[data-title-bar] [data-title-bar-command]')
const listButtonIds = (page: Page): Promise<string[]> =>
  getButtons(page).evaluateAll((all) =>
    all.map((one) => one.getAttribute('data-title-bar-command')!),
  )

test.describe('Plugin buttons in the title bar', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, [TREE_ID, 'enabled'], true)
  })

  test('runs the command of a button as the palette runs it', async ({ page }) => {
    await openWorkspace(page)
    await closeTab(page, /Close Files/i)
    await expect(getTab(page, 'Files')).toHaveCount(0)

    await getButton(page, TREE_BUTTON).click()

    await expect(getTab(page, 'Files')).toBeVisible()
  })

  test('shows the title and the shortcut of the command in the hover text', async ({ page }) => {
    await openWorkspace(page)

    const hover = await getButton(page, TREE_BUTTON).getAttribute('title')

    expect(hover).toMatch(/^Open file tree( \(.+\))?$/)
  })

  test('draws no pressed state on a button after a press', async ({ page }) => {
    await openWorkspace(page)

    await getButton(page, TREE_BUTTON).click()

    for (const name of ['aria-pressed', 'aria-checked', 'data-pressed', 'data-active']) {
      await expect(getButton(page, TREE_BUTTON)).not.toHaveAttribute(name, /.*/)
    }
  })

  test('opens the palette at the name of Create note and makes the note under the name typed', async ({
    page,
  }) => {
    await openWorkspace(page)

    await getButton(page, NOTE_BUTTON).click()

    await expect(getPaletteField(page)).toBeVisible()
    await getPaletteField(page).fill('Brine log')
    await page.keyboard.press('Enter')
    await expect(getTab(page, 'Brine log')).toBeVisible()
  })

  test('removes the button of a plugin that is disabled, and brings it back in its place', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    await expect(getButton(page, TREE_BUTTON)).toBeVisible()
    const before = await listButtonIds(page)
    expect(before).toContain(TREE_BUTTON)

    await writeSetting(request, [TREE_ID, 'enabled'], false)

    await expect(getButton(page, TREE_BUTTON)).toHaveCount(0)
    expect(await listButtonIds(page)).toStrictEqual(before.filter((one) => one !== TREE_BUTTON))

    await writeSetting(request, [TREE_ID, 'enabled'], true)

    await expect(getButton(page, TREE_BUTTON)).toHaveCount(1)
    expect(await listButtonIds(page)).toStrictEqual(before)
  })

  test('moves the buttons that do not fit into the menu in order, and a pick runs its command', async ({
    page,
  }) => {
    // The bar keeps 128px for the title and a button is 32px wide. The start area is the window
    // minus the three window controls (138px), so 380px leaves room for three slots: two buttons
    // and the menu button.
    await page.setViewportSize({ width: 380, height: 800 })
    await openWorkspace(page)
    await closeTab(page, /Close Agent/i)
    await expect(getTab(page, 'Agent')).toHaveCount(0)
    const more = page.locator('[data-title-bar-more]')
    await expect(more).toBeVisible()

    expect(await listButtonIds(page)).toStrictEqual([TREE_BUTTON, GRAPH_BUTTON])
    await more.click()
    const entries = page.getByRole('menuitem')
    await expect(entries).toHaveCount(3)
    expect(await entries.allTextContents()).toStrictEqual([
      'Open agent chat',
      'Create note',
      'Open settings',
    ])
    await entries.filter({ hasText: 'Open agent chat' }).click()
    await expect(getTab(page, 'Agent')).toBeVisible()
  })

  test('draws the same buttons in the window torn off from a tab', async ({ page }) => {
    await openWorkspace(page)
    await expect(getButton(page, GRAPH_BUTTON)).toBeVisible()
    const inMain = await listButtonIds(page)
    expect(inMain).toContain(GRAPH_BUTTON)

    await page.goto('/?window=secondary')

    await expect.poll(() => listButtonIds(page)).toStrictEqual(inMain)
  })

  test('draws the settings button last in the bar and a press opens the settings tab', async ({
    page,
  }) => {
    await openWorkspace(page)

    const ids = await listButtonIds(page)
    expect(ids.at(-1)).toBe(SETTINGS_BUTTON)
    await expect(getTab(page, 'Settings')).toHaveCount(0)

    await getButton(page, SETTINGS_BUTTON).click()

    await expect(getTab(page, 'Settings')).toBeVisible()
  })
})
