import { test, expect } from '../../support/test.js'
import { openWorkspace } from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNotePanel,
  getNoteText,
  openNote,
  writeNote,
} from '../../support/vault.js'

test.describe('Note editor and the frontmatter of the file', () => {
  test('shows the new prose alone when another program rewrites the open, unedited note', async ({
    page,
    request,
  }) => {
    const notePath = await createNote(request, createUniqueTitle('Frontmatter rewrite'))
    try {
      await writeNote(request, notePath, 'Old prose.')
      await openWorkspace(page)
      const text = await openNote(page, notePath)
      await expect(text).toHaveText('Old prose.')

      await writeNote(request, notePath, 'New prose.')

      await expect(getNoteText(page, notePath)).toHaveText('New prose.')
      await expect(getNotePanel(page, notePath).getByRole('alert')).toHaveCount(0)
    } finally {
      await destroyNote(request, notePath)
    }
  })
})
