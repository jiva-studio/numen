/** The rows of a setting of a plugin that holds connections: each row is a connection and a model. */
import { expect, type Locator, type Page } from 'playwright/test'
import { escapeRegExp } from '../rules.js'

const ROW = 'li.connection-rows__row'

export class ConnectionRows {
  constructor(private readonly page: Page) {}

  getRows(panel: Locator): Locator {
    return panel.locator(ROW)
  }

  /** A setting of one row shows its row at once; a setting of more rows adds one with Add connection. */
  async add(panel: Locator): Promise<Locator> {
    const addButton = panel.getByRole('button', { name: 'Add connection' })
    await expect(addButton.or(this.getRows(panel)).first()).toBeVisible()
    if ((await addButton.count()) === 0) {
      await expect(this.getRows(panel)).toHaveCount(1)
      return this.getRows(panel).first()
    }
    const before = await this.getRows(panel).count()
    await addButton.click()
    await expect(this.getRows(panel)).toHaveCount(before + 1)
    return this.getRows(panel).nth(before)
  }

  async chooseConnection(row: Locator, name: string): Promise<void> {
    await row.getByRole('button', { name: 'Connection', exact: true }).click()
    await this.page.getByRole('option', { name: new RegExp(`^${escapeRegExp(name)}`) }).click()
  }

  /** The model is searched in the list the service gives, and typed when the list does not hold it. */
  async chooseModel(row: Locator, model: string): Promise<void> {
    await row.getByRole('button', { name: 'Model', exact: true }).click()
    const search = this.page.getByRole('combobox', { name: 'Search model' })
    await search.fill(model)
    await search.press('Enter')
  }

  getQuestion(): Locator {
    return this.page.getByRole('alertdialog')
  }
}
