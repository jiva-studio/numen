import { test, expect, forgetOpenTabs, readActivePath, type Page } from '../../support/test.js'
import {
  getGraph,
  getPaletteField,
  openWorkspace,
  runCommand,
  waitForWindow,
} from '../../support/workspace.js'
import { openNote } from '../../support/vault.js'

const OPEN_NOTE = 'Open note'
const SHOW_IN_PLEX = 'Show in plex'
const SHOW_IN_FILE_TREE = 'Show in file tree'
/** A note of the vault the scenarios share, which the plex does not have in focus when the window opens. */
const NOTE = 'Entropy'
const CREATE_NOTE = `Create note “${NOTE}”`
const CREATE_ACTIONS = [
  CREATE_NOTE,
  `Create as child “${NOTE}”`,
  `Create as parent “${NOTE}”`,
  `Create as jump “${NOTE}”`,
]

const openWindow = async (page: Page): Promise<void> => {
  await forgetOpenTabs(page.request)
  await page.goto('/')
  await waitForWindow(page)
}

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

const openActions = async (page: Page) => {
  await page.keyboard.press('Control+k')
  const rows = page.locator('[data-actions="list"] [role="option"]')
  await expect(rows.first()).toBeVisible()
  return rows
}

const setPlexEnabled = async (page: Page, isEnabled: boolean): Promise<void> => {
  await runCommand(page, 'Open settings', /^Open settings$/)
  const settings = page.getByRole('tabpanel', { name: /Settings/ })
  await settings.getByRole('button', { name: /Plex Graph/ }).click()
  const toggle = settings.getByRole('switch', { name: /Enabled Plex Graph/ })
  if ((await toggle.isChecked()) !== isEnabled) await toggle.click()
  await expect(toggle).toBeChecked({ checked: isEnabled })
}

test.describe('The actions of a note found by its name, with the plex plugin enabled', () => {
  test('lists opening the note, showing it, then the Create actions', async ({ page }) => {
    await openWorkspace(page)

    await searchForNote(page)
    const rows = await openActions(page)

    await expect(rows.locator('[data-actions="name"]')).toHaveText([
      OPEN_NOTE,
      SHOW_IN_PLEX,
      SHOW_IN_FILE_TREE,
      ...CREATE_ACTIONS,
    ])
  })

  test('opens the note in a tab on Enter', async ({ page }) => {
    await openWorkspace(page)

    await searchForNote(page)
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${NOTE}\\.md$`))
  })

  test('shows the note in the plex when Show in plex is chosen', async ({ page }) => {
    await openWorkspace(page)
    await expect(getGraph(page).getByRole('img', { name: `${NOTE}, focus` })).toBeHidden()

    await searchForNote(page)
    const rows = await openActions(page)
    await rows.filter({ hasText: SHOW_IN_PLEX }).click()

    await expect(getGraph(page).getByRole('img', { name: `${NOTE}, focus` })).toBeAttached()
  })
})

test.describe('The command Show in plex over the note in front', () => {
  test.afterEach(async ({ page }) => {
    await openWindow(page)
    await setPlexEnabled(page, true)
  })

  test('is in the palette while the plex plugin is enabled and gone while it is disabled', async ({
    page,
  }) => {
    const options = (name: string) =>
      page.getByRole('option').filter({
        has: page.locator('[data-palette="name"]', { hasText: new RegExp(`^${name}$`) }),
      })
    const searchPalette = async (query: string) => {
      await page.keyboard.press('Control+p')
      await getPaletteField(page).fill(query)
    }

    await openWorkspace(page)
    await openNote(page, 'notes/Entropy.md')
    await searchPalette(SHOW_IN_PLEX)
    await expect(options(SHOW_IN_PLEX)).toHaveCount(1)
    await page.keyboard.press('Escape')

    await setPlexEnabled(page, false)
    await openNote(page, 'notes/Entropy.md')
    await searchPalette('Show in')
    await expect(options(SHOW_IN_FILE_TREE)).toHaveCount(1)
    await expect(options(SHOW_IN_PLEX)).toHaveCount(0)
  })
})

test.describe('The actions of a note found by its name, with the plex plugin disabled', () => {
  test.afterEach(async ({ page }) => {
    await openWindow(page)
    await setPlexEnabled(page, true)
  })

  test('lists no Show in plex, and Enter still opens the note in a tab', async ({ page }) => {
    await openWindow(page)
    await setPlexEnabled(page, false)

    await searchForNote(page)
    const rows = await openActions(page)
    await expect(rows.locator('[data-actions="name"]')).toHaveText([
      OPEN_NOTE,
      SHOW_IN_FILE_TREE,
      CREATE_NOTE,
    ])
    await page.keyboard.press('Escape')
    await page.keyboard.press('Enter')

    await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${NOTE}\\.md$`))
  })
})
