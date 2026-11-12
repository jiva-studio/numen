/** The page "Connections" of the Settings tab: the list of connections, the dialog that adds or edits one, the question before a delete. */
import { expect, type Locator, type Page } from 'playwright/test'
import { escapeRegExp } from '../rules.js'

export class ConnectionsPage {
  constructor(private readonly page: Page) {}

  /** The entry of the list of pages that opens the page. */
  getEntry(panel: Locator): Locator {
    return panel.getByRole('button', { name: /^Connections$/ })
  }

  getAddButton(panel: Locator): Locator {
    return panel.getByRole('button', { name: 'Add connection' })
  }

  getDialog(): Locator {
    return this.page.getByRole('dialog')
  }

  getRow(panel: Locator, name: string): Locator {
    return panel.getByRole('listitem', { name: new RegExp(`^${escapeRegExp(name)}$`) })
  }

  /** Chooses a type in the open dialog; the list of the types is a menu. */
  async chooseType(type: string): Promise<void> {
    await this.getDialog().locator('[data-slot="select"]').first().click()
    await this.page.getByRole('menuitemradio', { name: new RegExp(`^${escapeRegExp(type)}`) }).click()
  }

  async fill(label: string, value: string): Promise<void> {
    await this.getDialog().getByLabel(label, { exact: true }).fill(value)
  }

  async save(): Promise<void> {
    await this.getDialog().getByRole('button', { name: 'Save' }).click()
    await expect(this.getDialog()).toHaveCount(0)
  }

  async cancel(): Promise<void> {
    await this.getDialog().getByRole('button', { name: 'Cancel' }).click()
    await expect(this.getDialog()).toHaveCount(0)
  }

  /** Opens the dialog that edits the connection. */
  async openEdit(panel: Locator, name: string): Promise<void> {
    await this.getRow(panel, name).getByRole('button', { name: 'Connection actions' }).click()
    await this.page.getByRole('menuitem', { name: 'Edit' }).click()
    await expect(this.getDialog()).toBeVisible()
  }

  async chooseDelete(panel: Locator, name: string): Promise<void> {
    await this.getRow(panel, name).getByRole('button', { name: 'Connection actions' }).click()
    await this.page.getByRole('menuitem', { name: 'Delete' }).click()
  }
}
