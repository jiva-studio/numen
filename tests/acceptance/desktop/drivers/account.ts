/** The person at the card of the account on the page "Identity and licenses": deletes the cloud data, restores the account and exports what the server holds. */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator } from 'playwright/test'
import type { IdentityDriver } from './identity.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const SUBSCRIPTION_TITLE = 'Numen Sync'
const ACCOUNT_FINGERPRINT = 'Account fingerprint'
const DELETE_CLOUD_DATA = 'Delete cloud data'
const RESTORE_ACCOUNT = 'Restore account'
const EXPORT_ACCOUNT_DATA = 'Export account data'
const EXPORT_FILE = 'Numen Sync account data.json'
/** The plugin reads the account when the card opens and every 3 seconds while a payment is awaited. */
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

  /** Opens the question of the card, types the account fingerprint and confirms the deletion of the cloud data. */
  async deleteCloudData(): Promise<void> {
    await this.openDeleteQuestion()
    await this.confirmDeleteCloudData(this.identity.getAccountFingerprint())
  }

  /** Opens the question of the card. */
  async openDeleteQuestion(): Promise<void> {
    const card = await this.getCard()
    await card.getByRole('button', { name: DELETE_CLOUD_DATA }).click()
  }

  /** Types the text into the open question of the card and confirms the deletion. */
  async confirmDeleteCloudData(name: string): Promise<void> {
    const dialog = (await this.getCard()).getByRole('alertdialog')
    await dialog.getByRole('textbox', { name: ACCOUNT_FINGERPRINT }).fill(name)
    await dialog.getByRole('button', { name: DELETE_CLOUD_DATA }).click()
  }

  /** Expects the question of the card to say what is deleted and for how long, and what stays. */
  async expectDeleteQuestion(texts: readonly string[]): Promise<void> {
    const dialog = (await this.getCard()).getByRole('alertdialog')
    for (const text of texts) await expect(dialog.getByText(text, { exact: false }).first()).toBeVisible()
  }

  /** Expects the card to show that the cloud data is being deleted, with the action to restore. */
  async expectDeleting(isDeleting: boolean): Promise<void> {
    const card = await this.getCard()
    const restore = card.getByRole('button', { name: RESTORE_ACCOUNT })
    if (isDeleting) {
      await expect(card.locator('header').getByText('Deleting', { exact: true })).toBeVisible({ timeout: ACCOUNT_LIMIT_MS })
      await expect(restore).toBeVisible()
    } else {
      await expect(restore).toHaveCount(0, { timeout: ACCOUNT_LIMIT_MS })
    }
  }

  async restoreAccount(): Promise<void> {
    const card = await this.getCard()
    await card.getByRole('button', { name: RESTORE_ACCOUNT }).click()
  }

  /** Exports the account data into the folder the save dialog answers with, and keeps the text of the file. */
  async exportAccountData(): Promise<void> {
    const folder = path.join(this.scenario.getWorld().root, `account-export-${++this.downloads}`)
    fs.mkdirSync(folder, { recursive: true })
    const card = await this.getCard()
    const download = this.scenario.page.waitForEvent('download')
    await card.getByRole('button', { name: EXPORT_ACCOUNT_DATA }).click()
    const saved = await download
    expect(saved.suggestedFilename()).toBe(EXPORT_FILE)
    const target = path.join(folder, EXPORT_FILE)
    await saved.saveAs(target)
    this.exported = fs.readFileSync(target, 'utf8')
  }

  /** Expects the file the export saved to hold the account fingerprint. */
  expectExportNamesAccount(): void {
    expect(this.exported).toContain(this.identity.getAccountFingerprint())
  }
}
