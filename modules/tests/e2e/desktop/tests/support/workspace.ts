import { expect, type Locator, type Page } from 'playwright/test'

/** The graph of the note the vault opens with, which the workspace draws once the vault is read. */
export const getGraph = (page: Page): Locator => page.getByRole('group', { name: 'Neighbourhood' })

/** Loads the window and waits until the vault is read and the opening note is drawn. */
export const openWorkspace = async (page: Page): Promise<void> => {
  await page.goto('/')
  await expect(getGraph(page)).toBeVisible()
}

/** The row the keyboard stands on in the open palette. */
const getLitRowName = (page: Page): Locator =>
  page.locator('[role="option"][aria-selected="true"] [data-palette="name"]')

/** Opens the palette on a command and waits until the keyboard stands on it. */
const findPaletteCommand = async (page: Page, query: string, name: string): Promise<Locator> => {
  await page.keyboard.press('Control+p')
  const field = page.locator('input[data-palette="field"]').first()
  await expect(field).toBeVisible()
  await field.fill(query)
  await expect(getLitRowName(page)).toHaveText(name)
  return field
}

/** Runs a palette command once the keyboard stands on it. */
export const runPaletteCommand = async (page: Page, query: string, name: string): Promise<void> => {
  await findPaletteCommand(page, query, name)
  await page.keyboard.press('Enter')
}
