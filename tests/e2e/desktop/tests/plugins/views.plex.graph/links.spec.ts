import { test, expect, type Locator, type Page, type Route } from '../../support/test.js'
import { getGraph, openWorkspace } from '../../support/workspace.js'
import { createUniqueTitle } from '../../support/vault.js'
import {
  createNoteWithBody,
  destroyNotes,
  getConnectionSides,
  getRow,
  getTab,
  listPaths,
  showInFiles,
} from '../../support/layout.js'

const showInPlex = async (page: Page, path: string): Promise<void> => {
  await showInFiles(page, ['notes'])
  await getRow(page, path).click({ button: 'right' })
  await page.getByRole('menuitem', { name: 'Show in plex', exact: true }).click()
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

const waitForQuickLinkPanel = async (dialog: Locator): Promise<void> => {
  const frame = dialog.locator('[data-numen-searchable-list]')
  await frame.evaluate((el) =>
    Promise.all(el.getAnimations({ subtree: true }).map((animation) => animation.finished)),
  )
}

const holdSearches = async (page: Page): Promise<() => Promise<void>> => {
  const parked: Route[] = []
  let isHolding = true
  await page.route(SEARCH_NAMES, async (route) => {
    if (isHolding) parked.push(route)
    else await route.continue()
  })
  return async () => {
    isHolding = false
    for (const route of parked) await route.continue()
  }
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

  test('replaces row 1 in place with the note of exactly the typed title and links it without creating a note', async ({
    page,
    request,
  }) => {
    const before = await listPaths(request, 'notes')
    await openWorkspace(page)
    await showInPlex(page, quay)
    const release = await holdSearches(page)
    const dialog = await openQuickLink(page, 'Quay')
    const field = dialog.getByRole('combobox')
    await field.fill('Jetty')

    const options = dialog.getByRole('option')
    await expect(options).toHaveText(['Create "Jetty"'])
    const row = await options.first().boundingBox()

    await release()
    await expect(options.first()).toHaveText('Jetty')
    await expect(dialog.getByRole('option', { name: /^Create/ })).toHaveCount(0)
    await expect(options.first()).toHaveAttribute('aria-selected', 'true')
    expect(await options.first().boundingBox()).toEqual(row)

    await field.press('Enter')
    await expect
      .poll(async () => {
        const sides = await getConnectionSides(request, quay, jetty)
        return sides.first || sides.second
      })
      .toBe(true)
    expect(await listPaths(request, 'notes')).toEqual(before)
  })

  test('creates and links a note with a click on Create', async ({ page, request }) => {
    const title = createUniqueTitle('Wharf')
    const created = `notes/${title}.md`
    expect(await listPaths(request, 'notes')).not.toContain(created)
    try {
      await openWorkspace(page)
      await showInPlex(page, quay)
      const dialog = await openQuickLink(page, 'Quay')
      await dialog.getByRole('combobox').fill(title)
      const option = dialog.getByRole('option', { name: `Create "${title}"` })
      await expect(option).toBeVisible()
      await waitForQuickLinkPanel(dialog)
      await option.click()

      await expect.poll(async () => listPaths(request, 'notes')).toContain(created)
      await expect
        .poll(async () => {
          const sides = await getConnectionSides(request, quay, created)
          return sides.first || sides.second
        })
        .toBe(true)
    } finally {
      await destroyNotes(request, [created])
    }
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
