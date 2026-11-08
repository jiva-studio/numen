import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNoteWithBody,
  destroyNotes,
  dragRowOnto,
  getRow,
  getTab,
  listPaths,
  readOpenTabs,
  showInFiles,
} from '../../support/layout.js'
import { listBacklinks, readNote } from '../../support/vault.js'

const LEDGER_BODY = '# Tidewater ledger\n\nThe miller wrote the height of every spring tide.\n'

test.describe('Moving files in the files tab', () => {

  test('moves a note into a folder by dragging its row, and its open tab and links follow', async ({
    page,
    request,
  }) => {
    const ledger = await createNoteWithBody(request, 'Tidewater ledger', 'notes', LEDGER_BODY)
    const index = await createNoteWithBody(
      request,
      'Harbour index',
      'notes',
      '# Harbour index\n\nSee [[Tidewater ledger]].\n',
    )
    const moved = 'daily/Tidewater ledger.md'
    try {
      await openWorkspace(page)
      await showInFiles(page, ['notes'])
      await getRow(page, ledger).press('Enter')
      await expect(getTab(page, 'Tidewater ledger')).toBeVisible()
      await expect.poll(async () => listBacklinks(request, ledger)).toContain(index)

      await getTab(page, 'Files').click()
      await dragRowOnto(page, getRow(page, ledger), getRow(page, 'daily'))

      await expect.poll(async () => listPaths(request, 'daily')).toContain(moved)
      expect(await listPaths(request, 'notes')).not.toContain(ledger)
      expect((await readNote(request, moved)).body).toBe(LEDGER_BODY)
      await expect
        .poll(async () => (await readOpenTabs(request)).tabs.map((one) => one.path))
        .toContain(moved)
      await expect.poll(async () => listBacklinks(request, moved)).toContain(index)

      await getTab(page, 'Tidewater ledger').click()
      await expect(page.getByRole('tabpanel', { name: /Tidewater ledger/ })).toContainText(
        'The miller wrote the height of every spring tide.',
      )
    } finally {
      await destroyNotes(request, [ledger, moved, index])
    }
  })
})
