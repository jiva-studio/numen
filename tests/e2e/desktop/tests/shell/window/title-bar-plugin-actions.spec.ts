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
const NOTE_BUTTON = 'app:note.create'

const getButton = (page: Page, id: string) => page.locator(`[data-title-bar-action="${id}"]`)
const getButtons = (page: Page) => page.locator('[data-title-bar] [data-title-bar-action]')
const listButtonIds = (page: Page): Promise<string[]> =>
  getButtons(page).evaluateAll((all) =>
    all.map((one) => one.getAttribute('data-title-bar-action')!),
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
})
