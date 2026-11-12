/** The person at the connections: the page "Connections", and the rows of connections on the page of a plugin. */
import { expect, type Locator } from 'playwright/test'
import { ConnectionRows } from './components/ConnectionRows.js'
import { ConnectionsPage } from './components/ConnectionsPage.js'
import type { Scenario } from './scenario.js'
import type { ServiceDriver } from './services.js'
import type { SettingsDriver } from './settings.js'

const KEY_STORED = 'A key is stored'
const KEY_LABEL = 'API key'

export class ConnectionsDriver {
  private readonly page: ConnectionsPage
  private readonly rows: ConnectionRows

  constructor(
    scenario: Scenario,
    private readonly settings: SettingsDriver,
    private readonly services: ServiceDriver,
  ) {
    this.page = new ConnectionsPage(scenario.page)
    this.rows = new ConnectionRows(scenario.page)
  }

  async add(name: string, type: string, service: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.getAddButton(panel).click()
    await this.page.chooseType(type)
    await this.page.fill('Name', name)
    await this.page.fill('Address', this.services.getAddress(service))
    await this.page.save()
    await expect(this.page.getRow(panel, name)).toHaveCount(1)
  }

  async giveKey(name: string, key: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.openEdit(panel, name)
    await this.page.fill(KEY_LABEL, key)
    await this.page.save()
  }

  async removeKey(name: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.openEdit(panel, name)
    await this.page.getDialog().getByRole('button', { name: 'Remove key' }).click()
    await this.page.save()
  }

  async expectKey(name: string, has: boolean): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.openEdit(panel, name)
    await expect(this.page.getDialog().getByText(KEY_STORED)).toHaveCount(has ? 1 : 0)
    await this.page.cancel()
  }

  async delete(name: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.chooseDelete(panel, name)
    await this.rows.getQuestion().getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(this.page.getRow(panel, name)).toHaveCount(0)
  }

  async expectListed(name: string, has: boolean): Promise<void> {
    const panel = await this.settings.showConnections()
    await expect(this.page.getRow(panel, name)).toHaveCount(has ? 1 : 0)
  }

  async expectState(name: string, state: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await expect(this.page.getRow(panel, name)).toContainText(state)
  }

  private async showRows(plugin: string): Promise<Locator> {
    return this.settings.showPage(plugin)
  }

  async addRow(connection: string, model: string, plugin: string): Promise<void> {
    const panel = await this.showRows(plugin)
    const row = await this.rows.add(panel)
    await this.rows.chooseConnection(row, connection)
    await this.rows.chooseModel(row, model)
  }

  async removeRow(position: number, plugin: string): Promise<void> {
    await this.rows.remove(await this.showRows(plugin), position)
  }

  async expectRowCount(plugin: string, count: number): Promise<void> {
    await expect(this.rows.getRows(await this.showRows(plugin))).toHaveCount(count)
  }

  async answerQuestion(button: string): Promise<void> {
    await this.rows.getQuestion().getByRole('button', { name: button, exact: true }).click()
    await expect(this.rows.getQuestion()).toHaveCount(0)
  }

  async expectQuestion(text: string): Promise<void> {
    await expect(this.rows.getQuestion()).toContainText(text)
  }
}
