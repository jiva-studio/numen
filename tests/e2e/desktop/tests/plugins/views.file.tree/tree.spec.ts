import { test, expect, type Page } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createFolder,
  createNoteWithBody,
  destroyNotes,
  getRow,
  getTab,
  listPaths,
  readOpenTabs,
  showInFiles,
  trashFiles,
} from '../../support/layout.js'
import { readNote } from '../../support/vault.js'

const getNameField = (page: Page) => page.getByRole('tree').getByRole('textbox')

const chooseNewItem = async (page: Page, row: string, item: string): Promise<void> => {
  await getRow(page, row).click({ button: 'right' })
  // The submenu opens over the item that opens it once the pointer reaches it.
  const newItem = page.getByRole('menuitem', { name: 'New', exact: true })
  await newItem.focus()
  await newItem.press('ArrowRight')
  await page.getByRole('menuitem', { name: item, exact: true }).click()
}

const nameNewItem = async (page: Page, row: string, item: string, name: string) => {
  await openWorkspace(page)
  await showInFiles(page, [])
  await chooseNewItem(page, row, item)
  await expect(getNameField(page)).toBeFocused()
  await getNameField(page).fill(name)
  await getNameField(page).press('Enter')
}

test.describe('Naming, making and removing files in the files tab', () => {
  test('creates a folder in a folder from the New menu of its row, under the name typed', async ({
    page,
    request,
  }) => {
    await createFolder(request, 'Saltworks')
    try {
      await nameNewItem(page, 'Saltworks', 'New folder', 'Brine tanks')

      await expect
        .poll(async () => listPaths(request, 'Saltworks'))
        .toEqual(['Saltworks/Brine tanks'])
      await expect(getRow(page, 'Saltworks/Brine tanks')).toBeVisible()
    } finally {
      await trashFiles(request, ['Saltworks'])
    }
  })

  test('renames a note from its row menu, and its open tab follows the file', async ({
    page,
    request,
  }) => {
    const before = await createNoteWithBody(
      request,
      'Sluice log',
      'notes',
      '# Sluice log\n\nGate opened at dawn.\n',
    )
    const after = 'notes/Brine pump.md'
    try {
      await openWorkspace(page)
      await showInFiles(page, ['notes'])
      await getRow(page, before).press('Enter')
      await expect(getTab(page, 'Sluice log')).toBeVisible()

      await getTab(page, 'Files').click()
      await getRow(page, before).click({ button: 'right' })
      await page.getByRole('menuitem', { name: 'Rename', exact: true }).click()
      await expect(getNameField(page)).toHaveValue('Sluice log.md')
      await getNameField(page).fill('Brine pump.md')
      await getNameField(page).press('Enter')

      await expect.poll(async () => listPaths(request, 'notes')).toContain(after)
      expect(await listPaths(request, 'notes')).not.toContain(before)
      expect((await readNote(request, after)).body).toBe('# Sluice log\n\nGate opened at dawn.\n')
      await expect
        .poll(async () => (await readOpenTabs(request)).tabs.map((one) => one.path))
        .toContain(after)
      expect((await readOpenTabs(request)).tabs.map((one) => one.path)).not.toContain(before)
    } finally {
      await destroyNotes(request, [before, after])
    }
  })
})
