/** The person at the card "Synced vaults" of the settings of the plugin "Numen Sync" and at the command "Sync this vault": syncs vaults and reads their state. */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator } from 'playwright/test'
import { addVault, callRPC, listVaults } from '../../../e2e/desktop/tests/support/vault.js'
import type { ApplicationDriver } from './application.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PLUGIN_NAME = 'Numen Sync'
const SECTION = 'Synced vaults'
const SYNC = 'Sync this vault'
const STOP_SYNCING = 'Stop syncing'
const FROM_CLOUD = 'Get a vault from the cloud'
const CLOUD_VAULT = 'Vault in the cloud'
const DEVICE_VAULT = 'Vault on this device'
const GET = 'Get vault'
const UP_TO_DATE = 'Up to date'
/** The plugin reads the vaults when the card opens and asks the server every second: a vault is synced within this time. */
const SYNC_LIMIT_MS = 20_000

export class SyncVaultsDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
    private readonly application: ApplicationDriver,
  ) {}

  private async show(): Promise<Locator> {
    const panel = await this.settings.showPage(PLUGIN_NAME)
    await expect(panel.getByText(SECTION, { exact: true }).first()).toBeVisible()
    return panel
  }

  private getRow(panel: Locator, vault: string): Locator {
    return panel.getByRole('listitem').filter({ hasText: vault })
  }

  /** Runs the command "Sync this vault" on the top folder of the vault that is open. */
  /** Chooses "Stop syncing" in the menu of the row of the vault and answers the question with the same button. */
  async stopSyncingVault(vault: string): Promise<void> {
    const row = this.getRow(await this.show(), vault)
    await row.getByRole('button', { name: `Actions for ${vault}` }).click()
    await this.scenario.page.getByRole('menuitem', { name: STOP_SYNCING, exact: true }).click()
    await this.scenario.page.getByRole('alertdialog').getByRole('button', { name: STOP_SYNCING, exact: true }).click()
  }

  async connectNew(_vault: string): Promise<void> {
    await this.application.runCommand(SYNC)
  }

  private async openCloud(panel: Locator): Promise<Locator> {
    await panel.getByRole('button', { name: FROM_CLOUD, exact: true }).click({ timeout: SYNC_LIMIT_MS })
    return this.scenario.page.getByRole('dialog')
  }

  /** Opens the dialog "Get a vault from the cloud" and expects the vault of the identity in it, then closes it. */
  async expectCanConnectTo(vault: string, synced: string): Promise<void> {
    const dialog = await this.openCloud(await this.show())
    await expect(dialog.getByLabel(CLOUD_VAULT)).toContainText(synced, { timeout: SYNC_LIMIT_MS })
    await expect(dialog.getByLabel(DEVICE_VAULT)).toContainText(vault)
    await dialog.getByRole('button', { name: 'Cancel', exact: true }).click()
  }

  async connectExisting(vault: string, synced: string): Promise<void> {
    const dialog = await this.openCloud(await this.show())
    await dialog.getByLabel(CLOUD_VAULT).selectOption({ label: synced })
    await dialog.getByLabel(DEVICE_VAULT).selectOption({ label: vault })
    await dialog.getByRole('button', { name: GET, exact: true }).click()
  }

  /** Answers the question of the dialog: "Connect" gets the vault, "Cancel" goes back. */
  async answerConfirmation(answer: string): Promise<void> {
    const name = answer === 'Connect' ? GET : answer
    await this.scenario.page.getByRole('dialog').getByRole('button', { name, exact: true }).click({ timeout: SYNC_LIMIT_MS })
  }

  async expectAskedToConfirm(text: string): Promise<void> {
    await expect(this.scenario.page.getByRole('dialog').getByText(text, { exact: true })).toBeVisible({ timeout: SYNC_LIMIT_MS })
  }

  /** A vault that is got without a question shows as up to date, and no dialog is open. */
  async expectNotAsked(): Promise<void> {
    const panel = await this.show()
    await expect(panel.getByText(UP_TO_DATE, { exact: true }).first()).toBeVisible({ timeout: SYNC_LIMIT_MS })
    await expect(this.scenario.page.getByRole('dialog')).toHaveCount(0)
  }

  async expectNotConnected(vault: string): Promise<void> {
    const panel = await this.show()
    await expect(this.getRow(panel, vault)).toHaveCount(0, { timeout: SYNC_LIMIT_MS })
  }

  /** Expects the card to list the vault as synced. */
  async expectConnected(vault: string): Promise<void> {
    const panel = await this.show()
    await expect(this.getRow(panel, vault)).toBeVisible({ timeout: SYNC_LIMIT_MS })
  }

  async expectWarning(text: string): Promise<void> {
    await expect(this.scenario.page.getByText(text).first()).toBeVisible({ timeout: SYNC_LIMIT_MS })
  }

  /** Presses the button of an action in the row of a vault, as "Remove the files" of a paused vault. */
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
