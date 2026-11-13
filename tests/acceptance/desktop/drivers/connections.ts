/** The person at the connections: the page "Connections", and the rows of connections on the page of a plugin. */
import { expect, type Locator } from 'playwright/test'
import { ConnectionRows } from './components/ConnectionRows.js'
import { ConnectionsPage } from './components/ConnectionsPage.js'
import { Question } from './components/Question.js'
import { escapeRegExp } from './rules.js'
import type { Scenario } from './scenario.js'
import type { ServiceDriver } from './services.js'
import type { SettingsDriver } from './settings.js'

/** The key field shows this mask in place of a saved key and is empty without one. */
const KEY_SAVED = '••••••••'
const KEY_LABEL = 'API key'
/** The column of a table of rows that names the connection; every other column is a field of the row, by its title: "model" is the field "Model". */
const CONNECTION_COLUMN = 'connection'

export interface ConnectionEntry {
  readonly name: string
  readonly type: string
  readonly service: string
}

const readTable = (table: readonly (readonly string[])[]): Map<string, string>[] => {
  const [head = [], ...rows] = table
  return rows.map((row) => new Map(head.map((column, index) => [column.trim(), (row[index] ?? '').trim()])))
}

const requireCell = (row: Map<string, string>, column: string): string => {
  const cell = row.get(column)
  if (cell === undefined || cell === '') throw new Error(`the table has no "${column}" in a row: give the column "${column}"`)
  return cell
}

const getFieldTitle = (column: string): string => column.charAt(0).toUpperCase() + column.slice(1)

export const readConnectionEntries = (table: readonly (readonly string[])[]): ConnectionEntry[] =>
  readTable(table).map((row) => ({
    name: requireCell(row, 'name'),
    type: requireCell(row, 'type'),
    service: requireCell(row, 'service'),
  }))

export class ConnectionsDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
    private readonly services: ServiceDriver,
  ) {}

  /** The window of a scenario with a world is opened after the driver is made, so the page is read at each use. */
  private get page(): ConnectionsPage {
    return new ConnectionsPage(this.scenario.page)
  }

  private get rows(): ConnectionRows {
    return new ConnectionRows(this.scenario.page)
  }

  async add(name: string, type: string, service: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.getAddButton(panel).click()
    await this.page.chooseType(type)
    await this.page.fill('Name', name)
    await this.page.fill('Address', await this.services.getAddress(service))
    await this.page.save()
    await expect(this.page.getRow(panel, name)).toHaveCount(1)
  }

  async addAll(entries: readonly ConnectionEntry[]): Promise<void> {
    for (const entry of entries) await this.add(entry.name, entry.type, entry.service)
  }

  async giveKey(name: string, key: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.openEdit(panel, name)
    await this.page.fill(KEY_LABEL, key)
    await this.page.save()
  }

  /** Opens the dialog of the connection, which stays open. */
  async open(name: string): Promise<void> {
    await this.page.openEdit(await this.settings.showConnections(), name)
  }

  /** The open dialog shows the mask of a saved key, and the key itself is not in the field. */
  async expectKeySaved(): Promise<void> {
    const field = this.page.getDialog().getByLabel(KEY_LABEL, { exact: true })
    await expect(field).toHaveAttribute('placeholder', KEY_SAVED)
    await expect(field).toHaveValue('')
  }

  async delete(name: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await this.page.chooseDelete(panel, name)
    await new Question(this.scenario.page).answer('Delete', new RegExp(`^Delete "${escapeRegExp(name)}"`))
    await expect(this.page.getRow(panel, name)).toHaveCount(0)
  }

  /** The list has each connection of the table, with its type. */
  async expectListed(table: readonly (readonly string[])[]): Promise<void> {
    const panel = await this.settings.showConnections()
    for (const row of readTable(table)) {
      const listed = this.page.getRow(panel, requireCell(row, 'name'))
      await expect(listed).toHaveCount(1)
      await expect(listed).toContainText(requireCell(row, 'type'))
    }
  }

  async expectState(name: string, state: string): Promise<void> {
    const panel = await this.settings.showConnections()
    await expect(this.page.getRow(panel, name)).toContainText(state)
  }

  private async showSetting(setting: string, plugin: string): Promise<Locator> {
    const block = this.rows.getSetting(await this.settings.showPage(plugin), setting)
    await expect(block, `the plugin "${plugin}" has no setting "${setting}" of connections`).toHaveCount(1)
    return block
  }

  /** Makes the rows of the setting those of the table, in its order: the rows there are first are changed, the rest added or removed. */
  async setRows(setting: string, plugin: string, table: readonly (readonly string[])[]): Promise<void> {
    const block = await this.showSetting(setting, plugin)
    const wanted = readTable(table)
    for (const [index, cells] of wanted.entries()) {
      const row = index < (await this.rows.getRows(block).count()) ? this.rows.getRows(block).nth(index) : await this.rows.add(block)
      await this.rows.chooseConnection(row, requireCell(cells, CONNECTION_COLUMN))
      for (const [column, value] of cells) {
        if (column !== CONNECTION_COLUMN) await this.rows.chooseValue(row, getFieldTitle(column), value)
      }
    }
    while ((await this.rows.getRows(block).count()) > wanted.length) await this.rows.removeLast(block)
  }

  /** The setting has the rows of the table, in its order, and no other. */
  async expectRows(setting: string, plugin: string, table: readonly (readonly string[])[]): Promise<void> {
    const block = await this.showSetting(setting, plugin)
    const wanted = readTable(table)
    await expect(this.rows.getRows(block)).toHaveCount(wanted.length)
    for (const [index, cells] of wanted.entries()) {
      const row = this.rows.getRows(block).nth(index)
      for (const [column, value] of cells) {
        await expect(this.rows.getField(row, getFieldTitle(column))).toHaveText(value)
      }
    }
  }

  async expectRowCount(plugin: string, count: number): Promise<void> {
    await expect(this.rows.getRows(await this.settings.showPage(plugin))).toHaveCount(count)
  }
}
