import { expect, type Locator, type Page } from 'playwright/test'

/** The graph of the note the vault opens with, which the workspace draws once the vault is read. */
export const getGraph = (page: Page): Locator => page.getByRole('group', { name: 'Neighbourhood' })

/** Loads the window and waits until the vault is read and the opening note is drawn. */
export const openWorkspace = async (page: Page): Promise<void> => {
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
    await expect(passages.first()).toBeVisible({ timeout: 10_000 })
  }).toPass({ timeout })
  return (await findPassages(page, query, path)).first()
}
