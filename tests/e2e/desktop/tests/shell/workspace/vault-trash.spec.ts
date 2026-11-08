import { test, expect, type Page } from '../../support/test.js'
import {
  getGraph,
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
  listFilePaths,
  openNote,
  readNote,
  readVaultFile,
  writeNote,
} from '../../support/vault.js'

const getPlexNode = (page: Page, title: string, relation: string) =>
  getGraph(page).getByRole('button', { name: `${title}, ${relation}`, exact: true })

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
      await runPaletteCommand(page, 'Show note in plex', 'Show note in plex')
      await expect(getPlexNode(page, targetTitle, 'child')).toBeAttached()

      await openNote(page, target)
      await runPaletteCommand(page, 'Move note to trash', 'Move note to trash')

      await expect(getNoteTab(page, target)).toHaveCount(0)
      await expect(page.locator(`[data-tree-row="${target}"]`)).toHaveCount(0)
      await expect(getPlexNode(page, targetTitle, 'child')).toHaveCount(0)
      await expect(await openTaskList(page)).toContainText(`Broken links in: ${linker}`)
      expect((await readNote(request, target)).error).toBe('ERROR_CODE_NOT_FOUND')
      expect(await listFilePaths(request, 'notes')).not.toContain(target)
      expect(await findTrashed(request, `${targetTitle}.md`)).toHaveLength(1)
      expect(await readVaultFile(request, linker)).toContain(targetTitle)
    } finally {
      await destroyNote(request, linker)
      await destroyNote(request, target)
      await destroyTrashed(request, `${targetTitle}.md`)
    }
  })
})
