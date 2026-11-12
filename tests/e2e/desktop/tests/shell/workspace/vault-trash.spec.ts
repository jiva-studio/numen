import { test, expect, type Page } from '../../support/test.js'
import { getGraph, openWorkspace, runPaletteCommand } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  destroyTrashed,
  getNoteTab,
  openNote,
  writeNote,
} from '../../support/vault.js'

const getPlexNode = (page: Page, title: string, relation: string) =>
  getGraph(page).getByRole('button', { name: `${title}, ${relation}`, exact: true })

test.describe('Removing and renaming a note', () => {
  test('closes the tab of a removed note and takes it off the tree and the plex', async ({
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
      await runPaletteCommand(page, 'Move to trash', 'Move to trash')

      await expect(getNoteTab(page, target)).toHaveCount(0)
      await expect(page.locator(`[data-tree-row="${target}"]`)).toHaveCount(0)
      await expect(getPlexNode(page, targetTitle, 'child')).toHaveCount(0)
    } finally {
      await destroyNote(request, linker)
      await destroyNote(request, target)
      destroyTrashed(`${targetTitle}.md`)
    }
  })
})
