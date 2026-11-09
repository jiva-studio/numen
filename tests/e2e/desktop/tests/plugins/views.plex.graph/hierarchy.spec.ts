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
import { getRow, showInFiles } from '../../support/layout.js'

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
  await runPaletteCommand(page, 'Show in plex', 'Show in plex')
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

  test('makes a parent from the palette, joined above the note in front', async ({
    page,
    request,
  }) => {
    const childTitle = createUniqueTitle('Delta')
    const parentTitle = createUniqueTitle('River basin')
    const note = await createNote(request, childTitle)
    try {
      await openWorkspace(page)
      await openNote(page, note)

      await runPaletteCommand(page, 'Create parent note', 'Create parent note')
      await nameNote(page, parentTitle)

      const parent = await waitForMadeNote(page, parentTitle)
      madeNotes.push(parent)
      expect((await getConnection(request, note, parent)).role).toBe('ROLE_PARENT')
      await showInPlex(page, parentTitle)
      await expect(getPlexNode(page, childTitle, 'child')).toBeAttached()
    } finally {
      await destroyNote(request, note)
    }
  })

  test('makes a jump from the menu of the node in focus of the plex', async ({ page, request }) => {
    const fromTitle = createUniqueTitle('Tidal bore')
    const jumpTitle = createUniqueTitle('Severn')
    const from = await createNote(request, fromTitle)
    try {
      await openWorkspace(page)
      await openNote(page, from)
      await showInPlex(page, fromTitle)

      await getPlexNode(page, fromTitle, 'focus').click({ button: 'right' })
      await page.getByRole('menuitem', { name: 'Create jump note' }).click()
      await nameNote(page, jumpTitle)

      const jump = await waitForMadeNote(page, jumpTitle)
      madeNotes.push(jump)
      expect((await getConnection(request, from, jump)).role).toBe('ROLE_JUMP')
      await showInPlex(page, jumpTitle)
      await expect(getPlexNode(page, fromTitle, 'jump')).toBeAttached()
    } finally {
      await destroyNote(request, from)
    }
  })
})

test.describe('The Relation submenu of a row of the file tree', () => {
  test('makes a child of the note of the row under the name typed, joined below it', async ({
    page,
    request,
  }) => {
    const parentTitle = createUniqueTitle('Lagoon')
    const childTitle = createUniqueTitle('Reef flat')
    const parent = await createNote(request, parentTitle)
    let child = ''
    try {
      await openWorkspace(page)
      await showInFiles(page, ['notes'])
      await getRow(page, parent).click({ button: 'right' })
      const relation = page.getByRole('menuitem', { name: 'Relation', exact: true })
      await relation.focus()
      await relation.press('ArrowRight')
      await page.getByRole('menuitem', { name: 'Create child note', exact: true }).click()
      await nameNote(page, childTitle)

      child = await waitForMadeNote(page, childTitle)
      expect((await getConnection(request, parent, child)).role).toBe('ROLE_CHILD')
    } finally {
      for (const one of [parent, child].filter(Boolean)) await destroyNote(request, one)
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
