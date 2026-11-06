import { test, expect, forgetOpenTabs, readActivePath, type Page } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runCommand,
  waitForWindow,
} from '../../support/workspace.js'

const OPEN_NOTE = 'Open note'
const SHOW_IN_PLEX = 'Show note in plex'
/** A note of the vault the scenarios share, which the plex does not have in focus when the window opens. */
const NOTE = 'Entropy'

/** Opens the window without waiting for the plex, which a disabled plugin does not draw. */
const openWindow = async (page: Page): Promise<void> => {
  await forgetOpenTabs(page.request)
  await page.goto('/')
  await waitForWindow(page)
}

/** Searches for the note by its name and puts the highlight on its row. */
const searchForNote = async (page: Page) => {
  await page.keyboard.press('Control+k')
  const field = getPaletteField(page)
  await expect(field).toBeVisible()
  await field.fill(NOTE)
  const row = page
    .getByRole('option')
    .filter({ has: page.locator('[data-palette="name"]', { hasText: new RegExp(`^${NOTE}$`) }) })
    .first()
  await expect(row).toBeVisible()
  // The groups answer in any order, and a group answering while the keyboard steps can move the
  // highlight back to the first row. Once all have answered, the row is reached by stepping through
  // every row once from where the highlight is.
  await expect(page.getByRole('listbox').locator('[role="group"][aria-busy="true"]')).toHaveCount(0)
  const rows = page.getByRole('option')
  for (
    let step = 0;
    step < (await rows.count()) && (await row.getAttribute('aria-selected')) !== 'true';
    step += 1
  ) {
    await page.keyboard.press('ArrowDown')
  }
  await expect(row).toHaveAttribute('aria-selected', 'true')
}

/** Opens the panel of actions over the row in front and gives the rows of it. */
const openActions = async (page: Page) => {
  await page.keyboard.press('Control+k')
  const rows = page.locator('[data-actions="list"] [role="option"]')
  await expect(rows.first()).toBeVisible()
  return rows
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
  test('lists opening the note and then showing it in the plex', async ({ page }) => {
    await openWorkspace(page)

    await searchForNote(page)
    const rows = await openActions(page)

    await expect(rows.locator('[data-actions="name"]')).toHaveText([OPEN_NOTE, SHOW_IN_PLEX])
  })

  test('opens the note in a tab on Enter', async ({ page }) => {
    await openWorkspace(page)

    await searchForNote(page)
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${NOTE}\\.md$`))
  })

  test('shows the note in the plex when the second action is chosen', async ({ page }) => {
    await openWorkspace(page)
    await expect(getGraph(page).getByRole('img', { name: `${NOTE}, focus` })).toBeHidden()

    await searchForNote(page)
    const rows = await openActions(page)
    await rows.filter({ hasText: SHOW_IN_PLEX }).click()

    await expect(getGraph(page).getByRole('img', { name: `${NOTE}, focus` })).toBeAttached()
  })
})

test.describe('The actions of a note found by its name, with the plex plugin disabled', () => {
  test.afterEach(async ({ page }) => {
    await openWindow(page)
    await setPlexEnabled(page, true)
  })

  test('lists opening the note alone, and Enter still opens it in a tab', async ({ page }) => {
    await openWindow(page)
    await setPlexEnabled(page, false)

    await searchForNote(page)
    const rows = await openActions(page)
    await expect(rows.locator('[data-actions="name"]')).toHaveText([OPEN_NOTE])
    await page.keyboard.press('Escape')
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${NOTE}\\.md$`))
  })
})
