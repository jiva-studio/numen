import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNoteWithBody,
  destroyNotes,
  getRow,
  getTab,
  readOpenTabs,
  showInFiles,
} from '../../support/layout.js'

const readNoteTabPaths = async (page: Page): Promise<string[]> =>
  (await readOpenTabs(page.request)).tabs
    .filter((one) => one.kind === 'note')
    .map((one) => one.path)

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
})
