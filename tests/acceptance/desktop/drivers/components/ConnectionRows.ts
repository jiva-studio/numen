/** The rows of a setting of a plugin that holds connections: each row is a connection and the values of its fields, such as a model. */
import { expect, type Locator, type Page } from 'playwright/test'
import { escapeRegExp } from '../rules.js'

const SETTING = 'section.connection-rows'
const SETTING_NAME = '.settings__name'
const ROW = 'li.connection-rows__row'
const CONNECTION = 'Connection'

export class ConnectionRows {
  constructor(private readonly page: Page) {}

  /** The block of the setting with the title, which holds its rows and the button that adds one. */
  getSetting(panel: Locator, title: string): Locator {
    const name = this.page.locator(SETTING_NAME, { hasText: new RegExp(`^${escapeRegExp(title)}$`) })
    return panel.locator(SETTING).filter({ has: name })
  }

  getRows(setting: Locator): Locator {
    return setting.locator(ROW)
  }

  /** A setting of one row shows its row at once; a setting of more rows adds one with Add connection. */
  async add(setting: Locator): Promise<Locator> {
    const addButton = setting.getByRole('button', { name: 'Add connection' })
    await expect(addButton.or(this.getRows(setting)).first()).toBeVisible()
    if ((await addButton.count()) === 0) {
      await expect(this.getRows(setting)).toHaveCount(1)
      return this.getRows(setting).first()
    }
    const before = await this.getRows(setting).count()
    await addButton.click()
    await expect(this.getRows(setting)).toHaveCount(before + 1)
    return this.getRows(setting).nth(before)
  }

  /** Removes the last row with its button Remove row. */
  async removeLast(setting: Locator): Promise<void> {
    const count = await this.getRows(setting).count()
    await setting.getByRole('button', { name: `Remove row ${count}`, exact: true }).click()
    await expect(this.getRows(setting)).toHaveCount(count - 1)
  }

  async chooseConnection(row: Locator, name: string): Promise<void> {
    await this.getField(row, CONNECTION).click()
    await this.page.getByRole('option', { name: new RegExp(`^${escapeRegExp(name)}`) }).click()
  }

  /** A field of text, such as the model, is searched in the list the service gives, and typed when the list does not hold it. */
  async chooseValue(row: Locator, field: string, value: string): Promise<void> {
    await this.getField(row, field).click()
    const search = this.page.getByRole('combobox', { name: `Search ${field.toLowerCase()}` })
    const option = this.page.getByRole('menuitemradio', { name: value, exact: true })
    await expect(search.or(option).first()).toBeVisible()
    if ((await search.count()) === 0) {
      await option.click()
      return
    }
    await search.fill(value)
    await search.press('Enter')
  }

  /** The button of a field of a row: Connection, or a field of the connection such as Model; it reads the value chosen. */
  getField(row: Locator, field: string): Locator {
    return row.getByRole('button', { name: field, exact: true })
  }
}
