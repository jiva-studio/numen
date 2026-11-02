import { test, expect, type Page } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openTaskList,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  destroyTrashed,
  findTrashed,
  getNoteTab,
  isSyncingTitles,
  listFilePaths,
  openNote,
  readNote,
  readVaultFile,
  writeNote,
  writeSetting,
  writeVaultFile,
} from '../../support/vault.js'

const SYNC_SETTING = ['naming', 'sync_title_and_filename'] as const

/** The node of the plex drawn for a note, by its title and what it is to the note in focus. */
const getPlexNode = (page: Page, title: string, relation: string) =>
  getGraph(page).getByRole('button', { name: `${title}, ${relation}`, exact: true })

/** Renames the note in front through the palette's Change title step. */
const changeTitle = async (page: Page, from: string, to: string) => {
  await runPaletteCommand(page, 'Change title', 'Change title')
  const field = getPaletteField(page)
  await expect(field).toHaveValue(from)
  await field.fill(to)
  await page.keyboard.press('Enter')
}

test.describe('Removing and renaming a note', () => {
  test('moves a removed note to the trash, off the tree and the plex, and reports the notes left linking to it', async ({
    page,
    request,
  }) => {
    const targetTitle = createUniqueTitle('Tide table')
    const target = await createNote(request, targetTitle)
    const linker = await createNote(request, createUniqueTitle('Harbour log'), 'notes', [
      { to: target, role: 'ROLE_CHILD' },
    ])
    try {
      await writeNote(request, target, 'High water at six.\n')
      await openWorkspace(page)
      await openNote(page, linker)
      await runPaletteCommand(page, 'Show in plex', 'Show in plex')
      await expect(getPlexNode(page, targetTitle, 'child')).toBeAttached()

      await openNote(page, target)
      await runPaletteCommand(page, 'Remove note', 'Remove note')

      await expect(getNoteTab(page, target)).toHaveCount(0)
      await expect(page.locator(`[data-tree-row="${target}"]`)).toHaveCount(0)
      await expect(getPlexNode(page, targetTitle, 'child')).toHaveCount(0)
      await expect(await openTaskList(page)).toContainText(`Broken links in: ${linker}`)
      expect((await readNote(request, target)).error).toBe('ERROR_CODE_MISSING')
      expect(await listFilePaths(request, 'notes')).not.toContain(target)
      expect(await findTrashed(request, `${targetTitle}.md`)).toHaveLength(1)
      expect(await readVaultFile(request, linker)).toContain(targetTitle)
    } finally {
      await destroyNote(request, linker)
      await destroyNote(request, target)
      await destroyTrashed(request, `${targetTitle}.md`)
    }
  })

  test('destroys a note only once its exact title is typed back', async ({ page, request }) => {
    const title = createUniqueTitle('Ballast')
    const notePath = await createNote(request, title)
    try {
      await writeNote(request, notePath, 'Stones laid in the hold.\n')
      await openWorkspace(page)
      await openNote(page, notePath)

      await page.keyboard.press('Control+p')
      await getPaletteField(page).fill('Remove note')
      await expect(page.getByRole('option', { selected: true })).toContainText('Remove note')
      await page.keyboard.press('Shift+Enter')
      const field = getPaletteField(page)
      const destroy = page.getByRole('option', { name: new RegExp(`^Destroy “`) })
      await field.fill(`${title} wrong`)
      await expect(destroy).toHaveAttribute('aria-disabled', 'true')
      await page.keyboard.press('Enter')
      expect((await readNote(request, notePath)).body).toBe('Stones laid in the hold.\n')

      await field.fill(title)
      await expect(destroy).not.toHaveAttribute('aria-disabled', 'true')
      await page.keyboard.press('Enter')

      await expect(getNoteTab(page, notePath)).toHaveCount(0)
      await expect
        .poll(async () => (await readNote(request, notePath)).error)
        .toBe('ERROR_CODE_MISSING')
      expect(await findTrashed(request, `${title}.md`)).toHaveLength(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })

  test('renames the file and the links to it when the title changes while title and filename are synced', async ({
    page,
    request,
  }) => {
    const oldTitle = createUniqueTitle('Free energy')
    const newTitle = createUniqueTitle('Gibbs')
    const target = await createNote(request, oldTitle)
    const linker = await createNote(request, createUniqueTitle('Phase rule'), 'notes', [
      { to: target, role: 'ROLE_CHILD' },
    ])
    const renamedPath = `notes/${newTitle}.md`
    try {
      await writeSetting(request, SYNC_SETTING, true)
      await writeVaultFile(
        request,
        target,
        `---\ntitle: ${oldTitle}\n---\nThe work a system can do.\n`,
      )
      await openWorkspace(page)
      await openNote(page, target)

      await changeTitle(page, oldTitle, newTitle)

      await expect(getNoteTab(page, renamedPath)).toBeVisible()
      await expect.poll(() => listFilePaths(request, 'notes')).toContain(renamedPath)
      expect(await listFilePaths(request, 'notes')).not.toContain(target)
      expect((await readNote(request, renamedPath)).body).toContain('The work a system can do.')
      expect(await readVaultFile(request, renamedPath)).toMatch(
        new RegExp(`title: ["']?${newTitle}`),
      )
      const links = await readVaultFile(request, linker)
      expect(links).toContain(newTitle)
      expect(links).not.toContain(oldTitle)
    } finally {
      await destroyNote(request, linker)
      await destroyNote(request, target)
      await destroyNote(request, renamedPath)
    }
  })

  test('writes only the title into the frontmatter when title and filename are not synced', async ({
    page,
    request,
  }) => {
    const oldTitle = createUniqueTitle('Enthalpy')
    const newTitle = createUniqueTitle('Heat content')
    const target = await createNote(request, oldTitle)
    try {
      await writeSetting(request, SYNC_SETTING, true)
      await writeVaultFile(
        request,
        target,
        `---\ntitle: ${oldTitle}\n---\nHeat at constant pressure.\n`,
      )
      await openWorkspace(page)
      await openNote(page, target)

      await runPaletteCommand(page, 'Sync title and filename', 'Sync title and filename')
      await page.getByRole('option', { name: /^Off/ }).click()
      await expect.poll(() => isSyncingTitles(request)).toBe(false)
      await changeTitle(page, oldTitle, newTitle)

      await expect
        .poll(() => readVaultFile(request, target))
        .toMatch(new RegExp(`title: ["']?${newTitle}`))
      expect(await listFilePaths(request, 'notes')).toContain(target)
      expect(await listFilePaths(request, 'notes')).not.toContain(`notes/${newTitle}.md`)
      expect((await readNote(request, target)).body).toContain('Heat at constant pressure.')
    } finally {
      await writeSetting(request, SYNC_SETTING, true)
      await destroyNote(request, target)
    }
  })
})
