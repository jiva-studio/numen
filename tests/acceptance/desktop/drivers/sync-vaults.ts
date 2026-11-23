/** The person at the section "Vaults" of the settings of the plugin "Numen Sync": connects vaults to synced vaults and reads their state. */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator } from 'playwright/test'
import { addVault, callRPC, listVaults } from '../../../e2e/desktop/tests/support/vault.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PLUGIN_NAME = 'Numen Sync'
const SECTION = 'Vaults'
const CONNECT_NEW = 'Connect as a new synced vault'
const CONNECT_EXISTING = 'Connect to an existing synced vault'
const CHOOSE = 'Choose a synced vault'
const CONNECTED = 'Connected'
const NOT_CONNECTED = 'Not connected'
/** What the question of a connection says after its number. */
const CONFIRMATION = /differs? from the synced vault/
/** The plugin reads the vaults when the section opens and asks the server every second: a vault is connected within this time. */
const CONNECT_LIMIT_MS = 20_000

export class SyncVaultsDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
  ) {}

  private async show(): Promise<Locator> {
    const panel = await this.settings.showPage(PLUGIN_NAME)
    const section = panel.getByRole('tab', { name: SECTION, exact: true })
    if ((await section.count()) > 0) await section.click()
    await expect(panel.getByText(SECTION, { exact: true }).first()).toBeVisible()
    return panel
  }

  private getRow(panel: Locator, vault: string): Locator {
    return panel.getByRole('listitem').filter({ hasText: vault })
  }

  async connectNew(vault: string): Promise<void> {
    const row = this.getRow(await this.show(), vault)
    await row.getByRole('button', { name: CONNECT_NEW, exact: true }).click()
  }

  /** Opens the list of the synced vaults of the account for a vault and expects the synced vault in it. */
  async expectCanConnectTo(vault: string, synced: string): Promise<void> {
    const panel = await this.show()
    await this.getRow(panel, vault).getByRole('button', { name: CONNECT_EXISTING, exact: true }).click()
    await expect(panel.getByText(CHOOSE, { exact: true })).toBeVisible({ timeout: CONNECT_LIMIT_MS })
    await expect(panel.getByRole('button', { name: synced, exact: true })).toBeVisible()
  }

  async connectExisting(vault: string, synced: string): Promise<void> {
    await this.expectCanConnectTo(vault, synced)
    await (await this.show()).getByRole('button', { name: synced, exact: true }).click()
  }

  /** Chooses the synced vault and gives the answer to the question that asks for a confirmation. */
  async connectExistingAndAnswer(vault: string, synced: string, answer: string): Promise<void> {
    await this.connectExisting(vault, synced)
    const panel = await this.show()
    await panel.getByRole('button', { name: answer, exact: true }).click({ timeout: CONNECT_LIMIT_MS })
  }

  async expectAskedToConfirm(text: string): Promise<void> {
    const panel = await this.show()
    await expect(panel.getByText(text, { exact: true })).toBeVisible({ timeout: CONNECT_LIMIT_MS })
  }

  /** A vault that connects without a question shows as connected, and no question is on the page. */
  async expectNotAsked(): Promise<void> {
    const panel = await this.show()
    await expect(panel.getByText(CONNECTED, { exact: true }).first()).toBeVisible({ timeout: CONNECT_LIMIT_MS })
    await expect(panel.getByText(CONFIRMATION)).toHaveCount(0)
  }

  async expectNotConnected(vault: string): Promise<void> {
    const panel = await this.show()
    await expect(this.getRow(panel, vault).getByText(NOT_CONNECTED, { exact: true })).toBeVisible({ timeout: CONNECT_LIMIT_MS })
    await expect(panel.getByText(CONFIRMATION)).toHaveCount(0)
  }

  /** Expects the list to show the vault as connected. */
  async expectConnected(vault: string): Promise<void> {
    const panel = await this.show()
    await expect(this.getRow(panel, vault).getByText(CONNECTED, { exact: true })).toBeVisible({ timeout: CONNECT_LIMIT_MS })
  }

  async expectWarning(text: string): Promise<void> {
    const panel = await this.show()
    await expect(panel.getByText(text).first()).toBeVisible({ timeout: CONNECT_LIMIT_MS })
  }

  /** Presses the button of an action under the row of a vault, as "Remove the files" under a paused vault. */
  async choose(action: string): Promise<void> {
    const panel = await this.show()
    await panel.getByRole('button', { name: action, exact: true }).click()
  }

  async open(): Promise<void> {
    await this.show()
  }

  async rename(vault: string, name: string): Promise<void> {
    const { request } = this.scenario.page
    const found = (await listVaults(request)).find((one) => one.name === vault)
    if (found === undefined) throw new Error(`the list of vaults has no vault "${vault}"`)
    await callRPC(request, 'VaultsService/RenameVault', { id: found.id, name })
  }

  /** Makes a folder that has everything of a vault, including what the plugin keeps in it, and adds it to the list of vaults. */
  async addCopy(vault: string, copy: string): Promise<void> {
    const world = this.scenario.getWorld()
    const from = world.getFolder(this.scenario.getVault(vault))
    world.addVault(copy)
    fs.cpSync(from, world.getFolder(copy), { recursive: true })
    await addVault(this.scenario.page.request, path.resolve(world.getFolder(copy)), copy)
  }
}
