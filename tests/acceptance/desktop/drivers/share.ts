/** The person at the dialog "Share..." of a folder: adds a reader, reads the invitation, lists and removes readers. */
import { expect, type Locator } from 'playwright/test'
import type { ApplicationDriver } from './application.js'
import type { FileTreeDriver } from './file-tree.js'
import type { Scenario } from './scenario.js'

const SHARE = 'Share...'
const READER_CODE = 'Reader code'
const READER_FINGERPRINT = 'Reader account fingerprint'
const LABEL = 'Label'
const INVITATION = 'Invitation'
const READERS = 'Readers'
const STOP = 'Stop sharing'
/** The share is made and the reader is added within a few passes of the plugin. */
const SHARE_LIMIT_MS = 30_000

export interface GivenInvitation {
  readonly text: string
  readonly readerFingerprint: string
}

/** What the dialog did with a reader: gave the invitation, or refused in words. */
export type ShareOutcome = { readonly invitation: GivenInvitation } | { readonly refusal: string }

export class ShareDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly fileTree: FileTreeDriver,
    private readonly application: ApplicationDriver,
  ) {}

  private getDialog(): Locator {
    return this.scenario.page.getByRole('dialog')
  }

  /** Opens the dialog of the command "Share..." on the folder of the vault; the empty folder is the vault, asked through the palette. */
  async open(folder: string): Promise<Locator> {
    if (folder === '') await this.application.runCommand(SHARE)
    else await this.fileTree.runCommandOn(SHARE, 'folder', folder)
    const dialog = this.getDialog()
    await expect(dialog).toBeVisible()
    return dialog
  }

  async close(): Promise<void> {
    await this.scenario.page.keyboard.press('Escape')
    await expect(this.getDialog()).toHaveCount(0)
  }

  /**
   * Pastes the code of the reader, compares the fingerprint the dialog shows with the one the reader sees (the two people
   * compare it by voice), shares and returns the invitation the dialog shows.
   */
  async addReader(folder: string, code: string, readerFingerprint: string, label: string): Promise<ShareOutcome> {
    const dialog = await this.open(folder)
    await dialog.getByRole('textbox', { name: READER_CODE }).fill(code)
    await expect(dialog.getByLabel(READER_FINGERPRINT)).toHaveText(readerFingerprint)
    await dialog.getByRole('textbox', { name: LABEL }).fill(label)
    await dialog.getByRole('button', { name: 'Share', exact: true }).click()
    const invitation = dialog.getByRole('textbox', { name: INVITATION })
    const alert = dialog.getByRole('alert')
    await expect(invitation.or(alert)).toBeVisible({ timeout: SHARE_LIMIT_MS })
    const outcome: ShareOutcome = (await alert.isVisible())
      ? { refusal: (await alert.innerText()).trim() }
      : { invitation: { text: await invitation.inputValue(), readerFingerprint } }
    await this.close()
    return outcome
  }

  /** Tries to share and returns the words the dialog shows when it refuses, or undefined when it took the reader. */
  async tryAddReader(folder: string, code: string, label: string): Promise<string | undefined> {
    const dialog = await this.open(folder)
    await dialog.getByRole('textbox', { name: READER_CODE }).fill(code)
    await dialog.getByRole('textbox', { name: LABEL }).fill(label)
    await dialog.getByRole('button', { name: 'Share', exact: true }).click()
    const alert = dialog.getByRole('alert')
    await expect(alert).toBeVisible({ timeout: SHARE_LIMIT_MS })
    const text = (await alert.innerText()).trim()
    await this.close()
    return text
  }

  async expectReaderListed(folder: string, fingerprint: string): Promise<void> {
    const dialog = await this.open(folder)
    const readers = dialog.getByRole('list', { name: READERS })
    await expect(readers.getByRole('listitem').filter({ hasText: fingerprint })).toHaveCount(1, { timeout: SHARE_LIMIT_MS })
    await this.close()
  }

  async removeReader(folder: string, fingerprint: string): Promise<void> {
    const dialog = await this.open(folder)
    const row = dialog.getByRole('list', { name: READERS }).getByRole('listitem').filter({ hasText: fingerprint })
    await row.getByRole('button', { name: 'Remove', exact: true }).click()
    await expect(row).toHaveCount(0, { timeout: SHARE_LIMIT_MS })
    await this.close()
  }

  async stop(folder: string): Promise<void> {
    const dialog = await this.open(folder)
    await dialog.getByRole('button', { name: STOP, exact: true }).click()
    await this.close()
  }

  /** Expects the dialog of a refused share to show the words. */
  async expectTold(text: string): Promise<void> {
    await expect(this.scenario.page.getByText(text).first()).toBeVisible({ timeout: SHARE_LIMIT_MS })
  }
}
