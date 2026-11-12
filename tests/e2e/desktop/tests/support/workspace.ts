import { expect, type Locator, type Page } from 'playwright/test'
import { forgetOpenTabs } from './test.js'

/** The graph of the note the vault opens with, which the workspace draws once the vault is read. */
export const getGraph = (page: Page): Locator => page.getByRole('group', { name: 'Neighborhood' })

/**
 * Waits until the window is drawn: the task status of the title bar is there. It says "Idle" only
 * until the first list of tasks arrives, a moment after the page loads, and that list holds the
 * failed task of corrupted.pdf in every run, so the label cannot be waited for.
 */
export const waitForWindow = async (page: Page): Promise<void> => {
  await expect(page.locator('[data-tasks-status]')).toBeVisible()
}

/** Presses Enter on the row at `path` of the files tab once it is drawn: a folder opens, a file opens in a tab. */
export const openTreeRow = async (page: Page, path: string): Promise<void> => {
  const row = page.locator(`[data-tree-row="${path}"]`)
  await expect(row).toBeVisible()
  await row.press('Enter')
}

/** Loads the window and presses Enter on each row at `paths` of the files tab, in order. */
export const openFromFiles = async (page: Page, paths: readonly string[]): Promise<void> => {
  await page.goto('/')
  await waitForWindow(page)
  await page.getByRole('tab', { name: /^Files/ }).click()
  for (const path of paths) await openTreeRow(page, path)
}

/** Loads the window and opens the file `name` of the vault's `assets` folder from the files tab. */
export const openAsset = (page: Page, name: string): Promise<void> =>
  openFromFiles(page, ['assets', `assets/${name}`])

/** Presses the first close button named `name` once it is drawn. */
export const closeTab = async (page: Page, name: RegExp): Promise<void> => {
  const button = page.getByRole('button', { name }).first()
  await expect(button).toBeVisible()
  await button.click()
}

/**
 * Loads the window with the tabs a new vault opens with, and waits until the vault is read and the
 * opening note is drawn.
 */
export const openWorkspace = async (page: Page): Promise<void> => {
  await forgetOpenTabs(page.request)
  await page.goto('/')
  await expect(getGraph(page)).toBeVisible()
}

/** Runs the command of that exact name once the palette has drawn it for the words typed. */
export const runPaletteCommand = async (page: Page, query: string, name: string): Promise<void> => {
  await page.keyboard.press('Control+p')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(query)
  const command = page
    .getByRole('option')
    .filter({ has: page.locator('[data-palette="name"]', { hasText: new RegExp(`^${name}$`) }) })
    .first()
  await command.click()
}

/** The field the palette and the search are typed into. */
export const getPaletteField = (page: Page): Locator =>
  page.getByRole('combobox', { name: 'Search the vault' })

/** Runs the first command the palette offers for the words typed whose name matches `name`. */
export const runCommand = async (page: Page, query: string, name: RegExp): Promise<void> => {
  await page.keyboard.press('Control+p')
  await getPaletteField(page).fill(query)
  await page.getByRole('option', { name }).first().click()
}

/** Runs a command that offers items over the item the palette lists under the name `item`. */
export const runCommandOverItem = async (
  page: Page,
  query: string,
  name: RegExp,
  item: RegExp,
): Promise<void> => {
  await runCommand(page, query, name)
  await page.getByRole('option', { name: item }).first().click()
}

/**
 * Opens a file of the vault from the files tab and gives back the panel it is drawn in, found by
 * the file's name without its extension.
 */
export const openVaultFile = async (page: Page, folder: string, name: string): Promise<Locator> => {
  await page.getByRole('tab', { name: /^Files/ }).click()
  const parent = page.getByRole('treeitem', { name: folder, exact: true })
  if ((await parent.getAttribute('aria-expanded')) !== 'true') await parent.press('Enter')
  await page.getByRole('treeitem', { name, exact: true }).press('Enter')
  const stem = name.replace(/\.[^.]+$/, '').replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const panel = page.getByRole('tabpanel', { name: new RegExp(stem, 'i') })
  await expect(panel).toBeVisible()
  return panel
}

/** The passages the search finds for the words typed by their text, from the file at `path`. */
export const findPassages = async (page: Page, query: string, path: string): Promise<Locator> => {
  const field = getPaletteField(page)
  if (!(await field.isVisible())) await page.keyboard.press('Control+k')
  await field.fill('')
  await field.fill(query)
  return page.getByRole('group', { name: 'Text' }).getByRole('option').filter({ hasText: path })
}

/**
 * Waits until the search finds the words typed in the file at `path`, typing them again until it
 * does, and gives back the first passage.
 */
export const waitForPassage = async (
  page: Page,
  query: string,
  path: string,
  timeout = 90_000,
): Promise<Locator> => {
  await expect(async () => {
    const passages = await findPassages(page, query, path)
    await expect(passages.first()).toBeVisible({ timeout: 3_000 })
  }).toPass({ timeout })
  return (await findPassages(page, query, path)).first()
}

/** The window is usable within these milliseconds of loading, whatever a plugin or a download does. */
export const USABLE_WITHIN_MS = 5_000

/** The workspace is drawn: a tab bar, the panes, the plex graph of the opening note and the files. */
export const expectWorkspaceDrawn = async (page: Page): Promise<void> => {
  const within = { timeout: USABLE_WITHIN_MS }
  await expect(page.getByRole('tablist').first()).toBeVisible(within)
  await expect(page.locator('section[data-workspace-pane]')).toHaveCount(2, within)
  for (const pane of await page.locator('section[data-workspace-pane]').all()) {
    const box = await pane.boundingBox()
    expect(box?.width).toBeGreaterThan(0)
    expect(box?.height).toBeGreaterThan(0)
  }
  await expect(getGraph(page)).toBeVisible(within)
  await expect(page.getByRole('tab', { name: /^Files/ })).toBeVisible(within)
}

/** Opens the list of background tasks from the title bar, where a failed action and a warning are listed. */
export const openTaskList = async (page: Page): Promise<Locator> => {
  await page.locator('.tasks-status__trigger').click()
  const popover = page.locator('.task-popover')
  await expect(popover).toBeVisible()
  return popover
}
