import { test, expect } from '../../support/test.js'
import { createNote, createUniqueTitle } from '../../support/vault.js'
import { destroyNotes, getRow, showInFiles } from '../../support/layout.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'

test.describe('the names of the notes related to the related notes of a plex', () => {
  const notes: string[] = []

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, notes.splice(0))
  })

  test('shows the child of a child under the child, and moves the focus to it when clicked', async ({
    page,
    request,
  }) => {
    const root = createUniqueTitle('Orchard plan')
    const child = createUniqueTitle('Pear rows')
    const grandchild = createUniqueTitle('Rootstock list')
    const rootPath = await createNote(request, root)
    const childPath = await createNote(request, child, 'notes', [
      { to: rootPath, role: 'ROLE_PARENT' },
    ])
    const grandchildPath = await createNote(request, grandchild, 'notes', [
      { to: childPath, role: 'ROLE_PARENT' },
    ])
    notes.push(rootPath, childPath, grandchildPath)

    await openWorkspace(page)
    await showInFiles(page, ['notes'])
    await getRow(page, rootPath).press('Enter')

    const graph = getGraph(page)
    const name = graph.getByRole('button', { name: grandchild, exact: true })
    await expect(graph.getByLabel(`${child}, child`)).toBeVisible()
    await expect(name).toBeVisible()
    const nameBox = (await name.boundingBox())!
    const childBox = (await graph.getByLabel(`${child}, child`).boundingBox())!
    expect(nameBox.y).toBeGreaterThan(childBox.y + childBox.height / 2)
    await expect(graph.getByLabel(`${grandchild}, child`)).toHaveCount(0)

    await name.click()

    await expect(graph.getByLabel(`${grandchild}, focus`)).toBeVisible()
  })
})
