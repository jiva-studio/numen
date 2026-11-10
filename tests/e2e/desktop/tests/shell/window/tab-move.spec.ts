import { test, expect } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import {
  createNoteWithBody,
  destroyNotes,
  getRow,
  getTab,
  showInFiles,
} from '../../support/layout.js'

const WEIR = '# Weir notes\n\nThe weir holds the river at the height of the wheel.\n'
const FLUME = '# Flume notes\n\nThe flume carries the water onto the wheel.\n'

test.describe('Moving a tab to another window', () => {
  let weir = ''
  let flume = ''

  test.beforeEach(async ({ request }) => {
    weir = await createNoteWithBody(request, 'Weir notes', 'notes', WEIR)
    flume = await createNoteWithBody(request, 'Flume notes', 'notes', FLUME)
  })

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, [weir, flume])
  })

  test('opens the tab in the other window and closes it in this one', async ({
    page,
    context,
  }) => {
    await openWorkspace(page)
    const other = await context.newPage()
    await other.goto('/')
    await expect(getGraph(other)).toBeVisible()
    await showInFiles(other, ['notes'])
    await getRow(other, flume).press('Enter')
    await expect(getTab(other, 'Flume notes')).toBeVisible()

    await showInFiles(page, ['notes'])
    await getRow(page, weir).press('Enter')
    await expect(getTab(page, 'Weir notes')).toBeVisible()

    // The menu offers the other window once its tabs reach this window.
    const target = page.getByRole('menuitem', { name: /^Flume notes/ })
    await expect(async () => {
      await getTab(page, 'Weir notes').click({ button: 'right' })
      const moveItem = page.getByRole('menuitem', { name: 'Move to window', exact: true })
      await expect(moveItem).toBeVisible({ timeout: 1000 })
      await moveItem.focus()
      await moveItem.press('ArrowRight')
      await expect(target).toBeVisible({ timeout: 1000 })
    }).toPass()
    await target.click()

    await expect(getTab(page, 'Weir notes')).toHaveCount(0)
    await expect(getTab(other, 'Weir notes')).toBeVisible()
    await getTab(other, 'Weir notes').click()
    await expect(other.getByRole('tabpanel', { name: /Weir notes/ })).toContainText(
      'The weir holds the river at the height of the wheel.',
    )
  })
})
