import { test, expect, type Page } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import {
  createNoteWithBody,
  destroyNotes,
  getConnectionSides,
  getRow,
  getTab,
  showInFiles,
} from '../../support/layout.js'

const showInPlex = async (page: Page, path: string): Promise<void> => {
  await showInFiles(page, ['notes'])
  await getRow(page, path).click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Show note in plex', exact: true }).click()
}

const SEARCH_NAMES = '**/numen.v1.SearchService/SearchNames'

const openQuickLink = async (page: Page, title: string) => {
  await expect(getTab(page, title)).toBeVisible()
  await getGraph(page)
    .getByRole('img', { name: `${title}, focus` })
    .hover()
  const handle = page.locator('.plex__handle').first()
  await handle.hover()
  await handle.click()
  return page.getByRole('dialog', { name: 'New thought' })
}

test.describe('Links between two notes in the plex', () => {
  let quay = ''
  let jetty = ''

  test.beforeEach(async ({ request }) => {
    quay = await createNoteWithBody(
      request,
      'Quay',
      'notes',
      '# Quay\n\nWhere the barges unload.\n',
    )
    jetty = await createNoteWithBody(
      request,
      'Jetty',
      'notes',
      '# Jetty\n\nWhere the ferry ties up.\n',
    )
  })

  test.afterEach(async ({ request }) => {
    await destroyNotes(request, [quay, jetty])
  })

  test('links the focused note to a note found by its title with the quick link field', async ({
    page,
    request,
  }) => {
    expect(await getConnectionSides(request, quay, jetty)).toEqual({ first: false, second: false })
    await openWorkspace(page)
    await showInPlex(page, quay)
    const field = (await openQuickLink(page, 'Quay')).getByRole('combobox')
    await field.fill('Jett')
    await expect(page.getByRole('option', { name: 'Jetty', exact: true }).first()).toBeVisible()
    await field.press('ArrowDown')
    await field.press('Enter')

    await expect
      .poll(async () => {
        const sides = await getConnectionSides(request, quay, jetty)
        return sides.first || sides.second
      })
      .toBe(true)
    await expect(getGraph(page).getByRole('img', { name: /^(Quay|Jetty), focus$/ })).toBeVisible()
    await expect(getGraph(page).locator('.plex__edge')).toHaveCount(1)
  })
})
