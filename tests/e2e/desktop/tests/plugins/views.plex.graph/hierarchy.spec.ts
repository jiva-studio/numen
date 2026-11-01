import { test, expect, readActivePath, type Page } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runPaletteCommand,
} from '../../support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getConnection,
  getNoteTab,
  getNoteText,
  listFilePaths,
  openNote,
} from '../../support/vault.js'

/** The node of the plex drawn for a note, by its title and what it is to the note in focus. */
const getPlexNode = (page: Page, title: string, relation: string) =>
  getGraph(page).getByRole(relation === 'focus' ? 'img' : 'button', {
    name: `${title}, ${relation}`,
    exact: true,
  })

/** Names the note a command makes, in the palette's naming step. */
const nameNote = async (page: Page, title: string) => {
  const field = getPaletteField(page)
  await expect(field).toBeVisible()
  await field.fill(title)
  await expect(page.getByRole('option', { selected: true })).toContainText(title)
  await page.keyboard.press('Enter')
}

/** Waits until the note the window made is the tab in front, and gives back its path. */
const waitForMadeNote = async (page: Page, title: string): Promise<string> => {
  await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${title}\\.md$`))
  const made = await readActivePath(page.request)
  const folder = made.includes('/') ? made.slice(0, made.lastIndexOf('/')) : ''
  expect(await listFilePaths(page.request, folder)).toContain(made)
  return made
}

/** Puts the note in front into the focus of the plex. */
const showInPlex = async (page: Page, title: string) => {
  await runPaletteCommand(page, 'Show in plex', 'Show in plex')
  await expect(getPlexNode(page, title, 'focus')).toBeAttached()
}

test.describe('Making a related note', () => {
  test('makes a child from the palette, joined below the note in front', async ({
    page,
    request,
  }) => {
    const parentTitle = createUniqueTitle('Estuary')
    const childTitle = createUniqueTitle('Tributary')
    const parent = await createNote(request, parentTitle)
    let child = ''
    try {
      await openWorkspace(page)
      await openNote(page, parent)

      await runPaletteCommand(page, 'New child note', 'New child note')
      await nameNote(page, childTitle)

      child = await waitForMadeNote(page, childTitle)
      await expect(getNoteTab(page, child)).toHaveAttribute('aria-selected', 'true')
      expect((await getConnection(request, parent, child)).role).toBe('ROLE_CHILD')
      await showInPlex(page, childTitle)
      await expect(getPlexNode(page, parentTitle, 'parent')).toBeAttached()
    } finally {
      if (child) await destroyNote(request, child)
      await destroyNote(request, parent)
    }
  })

  test('makes a parent from the palette, joined above the note in front', async ({
    page,
    request,
  }) => {
    const childTitle = createUniqueTitle('Delta')
    const parentTitle = createUniqueTitle('River basin')
    const note = await createNote(request, childTitle)
    let parent = ''
    try {
      await openWorkspace(page)
      await openNote(page, note)

      await runPaletteCommand(page, 'New parent note', 'New parent note')
      await nameNote(page, parentTitle)

      parent = await waitForMadeNote(page, parentTitle)
      expect((await getConnection(request, note, parent)).role).toBe('ROLE_PARENT')
      await showInPlex(page, parentTitle)
      await expect(getPlexNode(page, childTitle, 'child')).toBeAttached()
    } finally {
      if (parent) await destroyNote(request, parent)
      await destroyNote(request, note)
    }
  })

  test('makes a jump from the menu of the node in focus of the plex', async ({
    page,
    request,
  }) => {
    const fromTitle = createUniqueTitle('Tidal bore')
    const jumpTitle = createUniqueTitle('Severn')
    const from = await createNote(request, fromTitle)
    let jump = ''
    try {
      await openWorkspace(page)
      await openNote(page, from)
      await showInPlex(page, fromTitle)

      await getPlexNode(page, fromTitle, 'focus').click({ button: 'right' })
      await page.getByRole('menuitem', { name: 'New jump note' }).click()
      await nameNote(page, jumpTitle)

      jump = await waitForMadeNote(page, jumpTitle)
      expect((await getConnection(request, from, jump)).role).toBe('ROLE_JUMP')
      await showInPlex(page, jumpTitle)
      await expect(getPlexNode(page, fromTitle, 'jump')).toBeAttached()
    } finally {
      if (jump) await destroyNote(request, jump)
      await destroyNote(request, from)
    }
  })
})

test.describe('A node of the plex', () => {
  test('opens its note in a tab when double-clicked', async ({ page }) => {
    await openWorkspace(page)

    await getPlexNode(page, 'Entropy', 'child').dblclick()

    await expect(getNoteTab(page, 'notes/Entropy.md')).toHaveAttribute('aria-selected', 'true')
    await expect(getNoteText(page, 'notes/Entropy.md')).toContainText('A measure of uncertainty')
  })
})
