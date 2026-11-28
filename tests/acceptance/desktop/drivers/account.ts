/** The person at the card of the identity on the page "Identity and licenses": deletes the cloud data and exports what the server holds. */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator } from 'playwright/test'
import type { IdentityDriver } from './identity.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const SUBSCRIPTION_TITLE = 'Numen Sync'
const MORE_ACTIONS = 'More actions'
const DELETE_CLOUD_DATA = 'Delete cloud data'
const CANCEL = 'Cancel'
const DELETED_BADGE = 'Deleted'
const DELETED_TEXT = 'The cloud data of this identity is deleted.'
const EXPORT_ACCOUNT_DATA = 'Export identity data'
const EXPORT_FILE = 'Numen Sync identity data.json'
/** The plugin reads the identity when the card opens and every 3 seconds while a payment is awaited. */
const ACCOUNT_LIMIT_MS = 20_000

export class AccountDriver {
  private exported = ''
  private downloads = 0

  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
    private readonly identity: IdentityDriver,
  ) {}

  private async getCard(): Promise<Locator> {
    const panel = await this.settings.showPage(PAGE)
    return panel.getByRole('region', { name: SUBSCRIPTION_TITLE, exact: true })
  }

  private async chooseFromMenu(card: Locator, item: string): Promise<void> {
    await card.getByRole('button', { name: MORE_ACTIONS }).click()
    await this.scenario.page.getByRole('menuitem', { name: item }).click()
  }

  /** Opens the question of the card and confirms the deletion of the cloud data. */
  async deleteCloudData(): Promise<void> {
    await this.openDeleteQuestion()
    await this.confirmDeleteCloudData()
  }

  /** Opens the question of the card. */
  async openDeleteQuestion(): Promise<void> {
    const card = await this.getCard()
    await this.chooseFromMenu(card, DELETE_CLOUD_DATA)
  }

  /** Confirms the deletion in the open question of the card. */
  async confirmDeleteCloudData(): Promise<void> {
    const dialog = (await this.getCard()).getByRole('alertdialog')
    await dialog.getByRole('button', { name: DELETE_CLOUD_DATA }).click()
  }

  /** Cancels the open question of the card. */
  async cancelDeleteCloudData(): Promise<void> {
    const dialog = (await this.getCard()).getByRole('alertdialog')
    await dialog.getByRole('button', { name: CANCEL }).click()
  }

  /** Expects the card to show that the cloud data is deleted and offer no way to restore it, or to keep its actions. */
  async expectDeleted(isDeleted: boolean): Promise<void> {
    const card = await this.getCard()
    const badge = card.locator('header').getByText(DELETED_BADGE, { exact: true })
    if (isDeleted) {
      await expect(badge).toBeVisible({ timeout: ACCOUNT_LIMIT_MS })
      await expect(card.getByText(DELETED_TEXT)).toBeVisible()
      await expect(card.getByRole('button', { name: /restore/i })).toHaveCount(0)
    } else {
      await expect(badge).toHaveCount(0, { timeout: ACCOUNT_LIMIT_MS })
      await expect(card.getByRole('button', { name: MORE_ACTIONS })).toBeVisible()
    }
  }

  /** Exports the identity data into the folder the save dialog answers with, and keeps the text of the file. */
  async exportAccountData(): Promise<void> {
    const folder = path.join(this.scenario.getWorld().root, `account-export-${++this.downloads}`)
    fs.mkdirSync(folder, { recursive: true })
    const card = await this.getCard()
    const download = this.scenario.page.waitForEvent('download')
    await this.chooseFromMenu(card, EXPORT_ACCOUNT_DATA)
    const saved = await download
    expect(saved.suggestedFilename()).toBe(EXPORT_FILE)
    const target = path.join(folder, EXPORT_FILE)
    await saved.saveAs(target)
    this.exported = fs.readFileSync(target, 'utf8')
  }

  /** Expects the file the export saved to hold the identity fingerprint. */
  expectExportNamesAccount(): void {
    expect(this.exported).toContain(this.identity.getAccountFingerprint())
  }
}
