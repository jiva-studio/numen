import { access } from 'node:fs/promises'
import { join } from 'node:path'
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
import { getVaultPath, readNote } from '../../support/vault.js'

const isOnDisk = async (vault: string, path: string): Promise<boolean> =>
  access(join(vault, path)).then(
    () => true,
    () => false,
  )

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
      await nameNewItem(page, 'Saltworks', 'Create folder', 'Brine tanks')

      await expect
        .poll(async () => listPaths(request, 'Saltworks'))
        .toEqual(['Saltworks/Brine tanks'])
      await expect(getRow(page, 'Saltworks/Brine tanks')).toBeVisible()
    } finally {
      await trashFiles(request, ['Saltworks'])
    }
  })

  test('creates a note in a folder from the New menu of its row, under the name typed', async ({
    page,
    request,
  }) => {
    await createFolder(request, 'Saltpans')
    try {
      await nameNewItem(page, 'Saltpans', 'Create note', 'Evaporation pans')

      await expect
        .poll(async () => listPaths(request, 'Saltpans'))
        .toEqual(['Saltpans/Evaporation pans.md'])
      expect((await readNote(request, 'Saltpans/Evaporation pans.md')).error).toBe('')
    } finally {
      await destroyNotes(request, ['Saltpans/Evaporation pans.md'])
      await trashFiles(request, ['Saltpans'])
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

  test('moves two notes selected with Ctrl-click to the trash with Move note to trash', async ({
    page,
    request,
  }) => {
    const first = await createNoteWithBody(request, 'Mill race', 'notes', '# Mill race\n')
    const second = await createNoteWithBody(request, 'Tail race', 'notes', '# Tail race\n')
    const headRace = await createNoteWithBody(request, 'Head race', 'notes', '# Head race\n')
    const vault = getVaultPath()
    try {
      await openWorkspace(page)
      await showInFiles(page, ['notes'])

      await getRow(page, first).click()
      await getRow(page, second).click({ modifiers: ['ControlOrMeta'] })
      await expect(getRow(page, first)).toHaveAttribute('aria-selected', 'true')
      await expect(getRow(page, second)).toHaveAttribute('aria-selected', 'true')
      await getRow(page, second).click({ button: 'right' })
      await page.getByRole('menuitem', { name: 'Move note to trash', exact: true }).click()

      await expect
        .poll(async () =>
          (await listPaths(request, 'notes')).filter((one) => one === first || one === second),
        )
        .toEqual([])
      expect(await listPaths(request, 'notes')).toContain(headRace)
      expect(await isOnDisk(vault, join('.trash', first))).toBe(true)
      expect(await isOnDisk(vault, join('.trash', second))).toBe(true)
      await expect(getRow(page, first)).toHaveCount(0)
      await expect(getRow(page, second)).toHaveCount(0)
    } finally {
      await destroyNotes(request, [first, second, headRace])
    }
  })

  test('persists folder expansion state across navigation', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['cards'])
    const animalsRow = getRow(page, 'cards/Animals.md')
    await expect(animalsRow).toBeVisible()

    await getTab(page, 'Agent').click()
    await getTab(page, 'Files').click()

    await expect(getRow(page, 'cards')).toHaveAttribute('aria-expanded', 'true')
    await expect(animalsRow).toBeVisible()
  })

  test('collapses expanded folder on Enter key', async ({ page }) => {
    await openWorkspace(page)
    await showInFiles(page, ['cards'])
    const cardsRow = getRow(page, 'cards')
    const animalsRow = getRow(page, 'cards/Animals.md')
    await expect(animalsRow).toBeVisible()

    await cardsRow.press('Enter')

    await expect(cardsRow).toHaveAttribute('aria-expanded', 'false')
    await expect(animalsRow).toBeHidden()
  })
})
