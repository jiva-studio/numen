import { test, expect, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import {
  createNoteWithBody,
  destroyNotes,
  getRow,
  getTab,
  readOpenTabs,
  showInFiles,
} from '../../support/layout.js'

/** The paths of the note tabs the core keeps for the vault in the window, in their order. */
const readNoteTabPaths = async (page: Page): Promise<string[]> =>
  (await readOpenTabs(page.request)).tabs
    .filter((one) => one.kind === 'note')
    .map((one) => one.path)

/** Links the tab of that title to the note tab of that title, from the Link with tab submenu. */
const linkTab = async (page: Page, from: string, to: string): Promise<void> => {
  await getTab(page, from).click({ button: 'right' })
  // The submenu opens over the item that opens it once the pointer reaches it.
  const linkItem = page.getByRole('menuitem', { name: 'Link with tab', exact: true })
  await linkItem.focus()
  await linkItem.press('ArrowRight')
  await page.getByRole('menuitem', { name: to, exact: true }).click()
  await expect(page.getByRole('tab', { name: new RegExp(`^Linked tab ${from} `) })).toBeVisible()
}

const WEIR = '# Weir notes\n\nThe weir holds the river at the height of the wheel.\n'
const FLUME = '# Flume notes\n\nThe flume carries the water onto the wheel.\n'

test.describe('Linked tabs', () => {
  let weir = ''
  let flume = ''

  test.beforeEach(async ({ request }) => {
    weir = await createNoteWithBody(request, 'Weir notes', 'notes', WEIR)
    flume = await createNoteWithBody(request, 'Flume notes', 'notes', FLUME)
  })

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, [weir, flume])
  })

  test('opens a note from the files tab in the note tab it is linked to', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, weir).press('Enter')
    await expect(getTab(page, 'Weir notes')).toBeVisible()
    await linkTab(page, 'Files', 'Weir notes')

    await getTab(page, 'Files').click()
    await getRow(page, flume).press('Enter')

    await expect.poll(() => readNoteTabPaths(page)).toEqual([flume])
    await expect(getTab(page, 'Flume notes')).toBeVisible()
    await expect(getTab(page, 'Weir notes')).toHaveCount(0)
    await getTab(page, 'Flume notes').click()
    await expect(page.getByRole('tabpanel', { name: /Flume notes/ })).toContainText(
      'The flume carries the water onto the wheel.',
    )
  })

  test('opens a note in a tab of its own once the files tab is unlinked', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, weir).press('Enter')
    await linkTab(page, 'Files', 'Weir notes')

    await getTab(page, 'Files').click({ button: 'right' })
    await page.getByRole('menuitem', { name: 'Unlink tab', exact: true }).click()
    await expect(getTab(page, 'Files')).toHaveAccessibleName('Files Close Files')

    await getTab(page, 'Files').click()
    await getRow(page, flume).press('Enter')

    await expect.poll(() => readNoteTabPaths(page)).toEqual([weir, flume])
    await expect(getTab(page, 'Weir notes')).toBeVisible()
    await expect(getTab(page, 'Flume notes')).toBeVisible()
  })

  test('drops the link when the linked note tab is closed', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, weir).press('Enter')
    await linkTab(page, 'Files', 'Weir notes')

    await page.getByRole('button', { name: 'Close Weir notes' }).click()
    await expect(getTab(page, 'Weir notes')).toHaveCount(0)
    await expect(getTab(page, 'Files')).toHaveAccessibleName('Files Close Files')

    await getTab(page, 'Files').click()
    await getRow(page, flume).press('Enter')
    await expect.poll(() => readNoteTabPaths(page)).toEqual([flume])
  })

  test('moves the note tab linked to the plex to the note chosen in the plex', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, weir).press('Enter')
    await expect(getTab(page, 'Weir notes')).toBeVisible()
    await linkTab(page, 'Thermodynamics', 'Weir notes')

    await getGraph(page).getByRole('button', { name: 'Entropy, child' }).click()

    await expect.poll(() => readNoteTabPaths(page)).toEqual(['notes/Entropy.md'])
    await expect(page.locator('[data-workspace-tab^="note:"]')).toHaveText(/Entropy/)
    await expect(getTab(page, 'Weir notes')).toHaveCount(0)
  })
})
