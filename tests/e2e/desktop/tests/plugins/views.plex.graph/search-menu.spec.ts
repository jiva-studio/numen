import { test, expect, readActivePath, type Page } from '../../support/test.js'
import { getGraph, getPaletteField, openWorkspace, runCommand } from '../../support/workspace.js'
import { createNote, createUniqueTitle, destroyNote } from '../../support/vault.js'

const OPEN_NOTE = 'Open note'
const SHOW_IN_PLEX = 'Show note in plex'

/** The row of the note in the result of the search, which the arrow keys and Enter act on. */
const searchForNote = async (page: Page, title: string) => {
  await page.keyboard.press('Control+k')
  const field = getPaletteField(page)
  await expect(field).toBeVisible()
  await field.fill(title)
  await expect(page.getByRole('option', { selected: true })).toContainText(title)
}

/** The names of the actions the panel over the row in front lists, in the order it lists them. */
const readActionNames = async (page: Page): Promise<string[]> => {
  await page.keyboard.press('Control+k')
  const rows = page.locator('[data-actions="list"] [role="option"]')
  await expect(rows.first()).toBeVisible()
  return rows.locator('[data-actions="name"]').allTextContents()
}

/** Turns the plugin on or off on its page of the settings. */
const setPlexEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /Plex Graph/ }).click()
  const toggle = settings.getByRole('switch', { name: /Enabled Plex Graph/ })
  if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('The actions of a note found by its name, with the plex plugin enabled', () => {
  const madeNotes: string[] = []

  test.afterEach(async ({ request }) => {
    for (const one of madeNotes.splice(0)) await destroyNote(request, one)
  })

  test('lists opening the note and then showing it in the plex', async ({ page }) => {
    const title = createUniqueTitle('Coral reef')
    madeNotes.push(await createNote(page.request, title))
    await openWorkspace(page)

    await searchForNote(page, title)

    expect(await readActionNames(page)).toStrictEqual([OPEN_NOTE, SHOW_IN_PLEX])
  })

  test('opens the note in a tab on Enter', async ({ page }) => {
    const title = createUniqueTitle('Coral reef')
    const made = await createNote(page.request, title)
    madeNotes.push(made)
    await openWorkspace(page)

    await searchForNote(page, title)
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toBe(made)
  })

  test('shows the note in the plex when the second action is chosen', async ({ page }) => {
    const title = createUniqueTitle('Coral reef')
    madeNotes.push(await createNote(page.request, title))
    await openWorkspace(page)

    await searchForNote(page, title)
    await readActionNames(page)
    await page.locator('[data-actions="list"] [role="option"]', { hasText: SHOW_IN_PLEX }).click()

    await expect(
      getGraph(page).getByRole('img', { name: `${title}, focus`, exact: true }),
    ).toBeAttached()
  })
})

test.describe('The actions of a note found by its name, with the plex plugin disabled', () => {
  const madeNotes: string[] = []

  test.afterEach(async ({ page, request }) => {
    for (const one of madeNotes.splice(0)) await destroyNote(request, one)
    await openWorkspace(page)
    await setPlexEnabled(page, true)
  })

  test('lists opening the note alone, and Enter still opens it in a tab', async ({ page }) => {
    const title = createUniqueTitle('Coral reef')
    const made = await createNote(page.request, title)
    madeNotes.push(made)
    await openWorkspace(page)
    await setPlexEnabled(page, false)

    await searchForNote(page, title)
    expect(await readActionNames(page)).toStrictEqual([OPEN_NOTE])
    await page.keyboard.press('Escape')
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toBe(made)
  })
})
