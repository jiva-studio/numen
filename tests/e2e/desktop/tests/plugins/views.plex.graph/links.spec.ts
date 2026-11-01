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

/** Shows the note at the path in the plex, from the menu on its row in the files tab. */
const showInPlex = async (page: Page, path: string): Promise<void> => {
  await showInFiles(page, ['notes'])
  await getRow(page, path).click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Show in plex', exact: true }).click()
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
    await expect(getTab(page, 'Quay')).toBeVisible()
    const focus = getGraph(page).getByRole('img', { name: 'Quay, focus' })
    await focus.hover()
    const handle = page.locator('.plex__handle').first()
    await handle.hover()
    await handle.click()

    const field = page.getByRole('dialog', { name: 'New thought' }).getByRole('textbox')
    await field.fill('Jett')
    await expect(page.getByRole('option', { name: 'Jetty', exact: true }).first()).toBeVisible()
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

  test('removes the link of two notes with Remove link, and the connection is gone from both', async ({
    page,
    request,
  }) => {
    await request.post('/numen.v1.NoteService/SaveConnection', {
      data: {
        connection: {
          first: quay,
          second: jetty,
          role: 'ROLE_CHILD',
          firstSide: { label: 'unloads at' },
        },
      },
    })
    expect(await getConnectionSides(request, quay, jetty)).toMatchObject({ first: true })
    await openWorkspace(page)
    await showInPlex(page, quay)
    await expect(getGraph(page).getByRole('img', { name: 'Quay, focus' })).toBeVisible()
    await expect(getGraph(page).locator('.plex__edge-hit')).toHaveCount(1)
    await expect(page.locator('.plex-frame:not([data-moving])')).toBeAttached()
    await getGraph(page)
      .locator('.plex__edge-hit')
      .first()
      .dispatchEvent('click', { clientX: 300, clientY: 300 })
    const popover = page.locator('.plex-link-popover')
    await expect(popover).toContainText('Quay')
    await expect(popover).toContainText('Jetty')

    await popover.getByRole('button', { name: 'Remove link' }).click()
    await expect(popover).toContainText('The link between these notes will be removed.')
    await popover.locator('.plex-link-popover__btn--done').click()

    await expect(popover).toBeHidden()
    await expect
      .poll(async () => getConnectionSides(request, quay, jetty))
      .toEqual({ first: false, second: false })
    expect(await getConnectionSides(request, jetty, quay)).toEqual({ first: false, second: false })
  })
})
