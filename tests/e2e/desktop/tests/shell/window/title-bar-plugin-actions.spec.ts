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

const getButton = (page: Page, id: string) => page.locator(`[data-title-bar-action="${id}"]`)
const getButtons = (page: Page) => page.locator('[data-title-bar] [data-title-bar-action]')
const listButtonIds = (page: Page): Promise<string[]> =>
  getButtons(page).evaluateAll((all) =>
    all.map((one) => one.getAttribute('data-title-bar-action')!),
  )

test.describe('Plugin buttons in the title bar', () => {
  test.afterEach(async ({ request }) => {
    await writeSetting(request, ['plugins', TREE_ID, 'enabled'], true)
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('runs the command of a button as the palette runs it', async ({ page }) => {
    await openWorkspace(page)
    await closeTab(page, /Close Files/i)
    await expect(getTab(page, 'Files')).toHaveCount(0)

    await getButton(page, TREE_BUTTON).click()

    await expect(getTab(page, 'Files')).toBeVisible()
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('shows the title and the shortcut of the command in the hover text', async ({
    page,
  }) => {
    await openWorkspace(page)

    const hover = await getButton(page, TREE_BUTTON).getAttribute('title')

    expect(hover).toMatch(/^Open file tree( \(.+\))?$/)
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('draws no pressed state on a button after a press', async ({ page }) => {
    await openWorkspace(page)

    await getButton(page, TREE_BUTTON).click()

    for (const name of ['aria-pressed', 'aria-checked', 'data-pressed', 'data-active']) {
      await expect(getButton(page, TREE_BUTTON)).not.toHaveAttribute(name, /.*/)
    }
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('opens the palette at the name of Create note and makes the note under the name typed', async ({
    page,
  }) => {
    await openWorkspace(page)

    await getButton(page, NOTE_BUTTON).click()

    await expect(getPaletteField(page)).toBeVisible()
    await getPaletteField(page).fill('Brine log')
    await page.keyboard.press('Enter')
    await expect(getTab(page, 'Brine log')).toBeVisible()
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('removes the button of a plugin that is disabled, and brings it back in its place', async ({
    page,
    request,
  }) => {
    await openWorkspace(page)
    const before = await listButtonIds(page)
    expect(before).toContain(TREE_BUTTON)

    await writeSetting(request, ['plugins', TREE_ID, 'enabled'], false)

    await expect(getButton(page, TREE_BUTTON)).toHaveCount(0)
    expect(await listButtonIds(page)).toStrictEqual(before.filter((one) => one !== TREE_BUTTON))

    await writeSetting(request, ['plugins', TREE_ID, 'enabled'], true)

    await expect(getButton(page, TREE_BUTTON)).toHaveCount(1)
    expect(await listButtonIds(page)).toStrictEqual(before)
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('moves the buttons that do not fit into the menu in order, and a pick runs its command', async ({
    page,
  }) => {
    await page.setViewportSize({ width: 520, height: 800 })
    await openWorkspace(page)
    await closeTab(page, /Close Files/i)
    const more = page.locator('[data-title-bar-more]')
    await expect(more).toBeVisible()
    const inBar = await listButtonIds(page)

    await more.click()
    const entries = page.getByRole('menuitem')
    const names = await entries.allTextContents()

    expect(names.length).toBeGreaterThan(0)
    expect(inBar).not.toContain(names[0])
    await entries.filter({ hasText: 'Open file tree' }).click()
    await expect(getTab(page, 'Files')).toBeVisible()
  })

  // #1507: waits for a plugin that places a command in titleBar; enable when the placements land.
  test.fixme('draws the same buttons in the window torn off from a tab', async ({ page }) => {
    await openWorkspace(page)
    const inMain = await listButtonIds(page)
    expect(inMain).toContain(GRAPH_BUTTON)

    await page.goto('/?window=secondary')

    await expect.poll(() => listButtonIds(page)).toStrictEqual(inMain)
  })
})
