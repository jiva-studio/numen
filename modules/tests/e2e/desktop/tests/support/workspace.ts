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
