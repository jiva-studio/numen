import { test, expect, readActivePath, type Page } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import { createNote, createUniqueTitle, destroyNote, getConnection, getNoteTab, listFilePaths, openNote } from '../../support/vault.js'

const getPlexNode = (page: Page, title: string, relation: string) =>
  getGraph(page).getByRole(relation === 'focus' ? 'img' : 'button', {
    name: `${title}, ${relation}`,
    exact: true,
  })

const nameNote = async (page: Page, title: string) => {
  const field = getPaletteField(page)
  await expect(field).toBeVisible()
  await field.fill(title)
  await expect(page.getByRole('option', { selected: true })).toContainText(title)
  await page.keyboard.press('Enter')
}

const waitForMadeNote = async (page: Page, title: string): Promise<string> => {
  await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${title}\\.md$`))
  const made = await readActivePath(page.request)
  const folder = made.includes('/') ? made.slice(0, made.lastIndexOf('/')) : ''
  expect(await listFilePaths(page.request, folder)).toContain(made)
  return made
}

const showInPlex = async (page: Page, title: string) => {
  await runPaletteCommand(page, 'Show note in plex', 'Show note in plex')
  await expect(getPlexNode(page, title, 'focus')).toBeAttached()
}

test.describe('Making a related note', () => {
  const madeNotes: string[] = []

  test.afterEach(async ({ request }) => {
    for (const one of madeNotes.splice(0)) await destroyNote(request, one)
  })

  test('makes a child from the palette, joined below the note in front', async ({
    page,
    request,
  }) => {
    const parentTitle = createUniqueTitle('Estuary')
    const childTitle = createUniqueTitle('Tributary')
    const parent = await createNote(request, parentTitle)
    try {
      await openWorkspace(page)
      await openNote(page, parent)

      await runPaletteCommand(page, 'Create child note', 'Create child note')
      await nameNote(page, childTitle)

      const child = await waitForMadeNote(page, childTitle)
      madeNotes.push(child)
      await expect(getNoteTab(page, child)).toHaveAttribute('aria-selected', 'true')
      expect((await getConnection(request, parent, child)).role).toBe('ROLE_CHILD')
      await showInPlex(page, childTitle)
      await expect(getPlexNode(page, parentTitle, 'parent')).toBeAttached()
    } finally {
      await destroyNote(request, parent)
    }
  })
})
