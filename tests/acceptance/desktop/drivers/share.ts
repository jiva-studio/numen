/** The person at the dialog "Share..." of a folder: gets the link, allows the device that asks, and lists and removes the devices that read. */
import { expect, type Locator } from 'playwright/test'
import type { ApplicationDriver } from './application.js'
import type { FileTreeDriver } from './file-tree.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const SHARE = 'Share...'
const LINK = 'Invitation link'
const PLUGIN_NAME = 'Numen Sync'
const PAGE = 'Shared folders'
const SHARED_BY_YOU = 'Shared by you'
const REMOVE = 'Remove access'
/** The device asks within a few passes of the plugin on both computers. */
const SHARE_LIMIT_MS = 30_000
const FINGERPRINT_GROUPS = 3

/** The first groups of a fingerprint, as the dialog and the page show a device. */
export const getShortFingerprint = (fingerprint: string): string =>
  fingerprint.trim().split(/\s+/).slice(0, FINGERPRINT_GROUPS).join(' ')

export class ShareDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly fileTree: FileTreeDriver,
    private readonly application: ApplicationDriver,
    private readonly settings: SettingsDriver,
  ) {}

  private getDialog(): Locator {
    return this.scenario.page.getByRole('dialog')
  }

  private async open(folder: string): Promise<Locator> {
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

  /** Opens the dialog and returns the link it shows, or the words it refuses with. The dialog stays open. */
  async start(folder: string): Promise<{ readonly link: string } | { readonly refusal: string }> {
    const dialog = await this.open(folder)
    const link = dialog.getByRole('textbox', { name: LINK })
    const alert = dialog.getByRole('alert')
    await expect(link.or(alert)).toBeVisible({ timeout: SHARE_LIMIT_MS })
    if (await alert.isVisible()) return { refusal: (await alert.innerText()).trim() }
    return { link: await link.inputValue() }
  }

  private async expectRequest(fingerprint: string, code: string): Promise<void> {
    const request = this.getDialog().getByRole('alert')
    await expect(request).toContainText(getShortFingerprint(fingerprint), {
      timeout: SHARE_LIMIT_MS,
    })
    await expect(request).toContainText(`Code ${code}`)
    await expect(request).toContainText(fingerprint)
  }

  /** Compares the code and the fingerprint of the device with the ones in the dialog, confirms the fingerprint, then allows the device and expects it among the devices that read. */
  async allow(fingerprint: string, code: string): Promise<void> {
    await this.expectRequest(fingerprint, code)
    const dialog = this.getDialog()
    await dialog.getByRole('checkbox', { name: 'The fingerprint is the same' }).check()
    await dialog.getByRole('button', { name: 'Approve', exact: true }).click()
    await expect(dialog.getByText(getShortFingerprint(fingerprint))).toBeVisible({
      timeout: SHARE_LIMIT_MS,
    })
    await this.close()
  }

  async decline(fingerprint: string, code: string): Promise<void> {
    await this.expectRequest(fingerprint, code)
    await this.getDialog().getByRole('button', { name: 'Decline', exact: true }).click()
    await this.close()
  }

  private async showShared(): Promise<Locator> {
    const panel = await this.settings.showPage(PLUGIN_NAME)
    const section = panel.getByRole('tab', { name: PAGE, exact: true })
    if ((await section.count()) > 0) await section.click()
    return panel
  }

  private async getDevice(folder: string, fingerprint: string): Promise<Locator> {
    const list = (await this.showShared()).getByRole('list', { name: SHARED_BY_YOU })
    const given = list.getByRole('listitem').filter({ hasText: folder }).first()
    return given.getByRole('listitem').filter({ hasText: getShortFingerprint(fingerprint) })
  }

  async expectReaderListed(folder: string, fingerprint: string): Promise<void> {
    await expect(await this.getDevice(folder, fingerprint)).toBeVisible({ timeout: SHARE_LIMIT_MS })
  }

  async removeReader(folder: string, fingerprint: string): Promise<void> {
    const device = await this.getDevice(folder, fingerprint)
    await device.getByRole('button', { name: REMOVE, exact: true }).click()
    await expect(device.getByRole('button', { name: REMOVE, exact: true })).toHaveCount(0, {
      timeout: SHARE_LIMIT_MS,
    })
  }
}
