/**
 * The person at the devices of the page "Identity and licenses" of one computer: the list of devices, the dialog that
 * shows the QR code that links a new device, the dialog that joins an identity by a QR code, and the revoke of a device.
 */
import { expect, type Locator } from 'playwright/test'
import { decodeQr } from './qr.js'
import { readFingerprint } from './link.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const LIST = 'Devices'
const LINK_NEW = 'Link a new device'
const JOIN_WITH_QR = 'Use existing'
const QR_IMAGE = 'QR code'
const PASTE_FIELD = 'Link from the other device'
const USE_LINK = 'Use link'
const LINK_THIS = 'Link device'
const DECLINE = "Don't link"
const JOIN = 'Use identity'
const REVOKE = 'Remove'
const THIS_DEVICE = 'This device'
const REVOKED = 'Revoked'
const DELETE_REVOKED = 'Delete identity from this device'
const ACCOUNT_FINGERPRINT = 'Identity fingerprint'
/** The people at the two computers act within seconds; the plugin carries a message in about a second. */
const STEP_LIMIT_MS = 30_000

/** A device as the list shows it. */
export interface DeviceRow {
  readonly fingerprint: string
  readonly isThis: boolean
  readonly isRevoked: boolean
}

export class DevicesDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
  ) {}

  private getDialog(): Locator {
    return this.scenario.page.getByRole('dialog')
  }

  private async showPage(): Promise<Locator> {
    await this.closeDialog()
    return this.settings.showPage(PAGE)
  }

  /** Closes the dialog of linking where one is open, so the page behind it can be used. */
  async closeDialog(): Promise<void> {
    const dialog = this.getDialog()
    if ((await dialog.count()) === 0) return
    await this.scenario.page.keyboard.press('Escape')
    await expect(dialog).toHaveCount(0)
  }

  private async openLinkDialog(): Promise<Locator> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: LINK_NEW, exact: true }).click()
    return this.getDialog()
  }

  private async openJoinDialog(): Promise<Locator> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: JOIN_WITH_QR, exact: true }).click()
    return this.getDialog()
  }

  /** Expects the page of this computer to offer no way to join an identity: it has one already. */
  async expectNoJoin(): Promise<void> {
    const panel = await this.showPage()
    await expect(panel.getByRole('button', { name: LINK_NEW, exact: true })).toBeVisible({ timeout: STEP_LIMIT_MS })
    await expect(panel.getByRole('button', { name: JOIN_WITH_QR, exact: true })).toHaveCount(0)
  }

  private async readQr(dialog: Locator): Promise<string> {
    const image = dialog.getByRole('img', { name: QR_IMAGE })
    await expect(image).toBeVisible({ timeout: STEP_LIMIT_MS })
    return decodeQr(this.scenario.page.context(), await image.screenshot())
  }

  /** Shows the QR code of a device that has the identity and returns the text the picture holds. */
  async showQrToLink(): Promise<string> {
    return this.readQr(await this.openLinkDialog())
  }

  /** A new device reads the QR code of a device that has the identity: the desktop takes the link by paste. */
  async scanToJoin(text: string): Promise<void> {
    const dialog = await this.openJoinDialog()
    const field = dialog.getByRole('textbox', { name: PASTE_FIELD })
    await expect(field).toBeVisible({ timeout: STEP_LIMIT_MS })
    await field.fill(text)
    await dialog.getByRole('button', { name: USE_LINK, exact: true }).click()
  }

  async approve(): Promise<void> {
    await this.getDialog().getByRole('button', { name: LINK_THIS, exact: true }).click({ timeout: STEP_LIMIT_MS })
  }

  async decline(): Promise<void> {
    await this.getDialog().getByRole('button', { name: DECLINE, exact: true }).click({ timeout: STEP_LIMIT_MS })
  }

  /** Expects the page of a new device to ask whether to join the identity with this fingerprint, and answers Join where answer is set. */
  async expectAskedToJoin(account: string, answer: boolean): Promise<void> {
    const dialog = this.getDialog()
    await expect(dialog.getByText(`Use identity ${account} on this device?`)).toBeVisible({ timeout: STEP_LIMIT_MS })
    if (answer) await dialog.getByRole('button', { name: JOIN, exact: true }).click()
  }

  /** Joins where the page asks to, and expects the page to show the identity. */
  async expectAccount(account: string): Promise<void> {
    await this.joinWhereAsked()
    const panel = await this.showPage()
    const shown = panel.locator('dt', { hasText: ACCOUNT_FINGERPRINT }).locator('xpath=following-sibling::dd[1]')
    await expect.poll(async () => (await shown.count()) > 0 && (await shown.innerText()).trim(), { timeout: STEP_LIMIT_MS }).toBe(account)
  }

  private async joinWhereAsked(): Promise<void> {
    const dialog = this.getDialog()
    const join = dialog.getByRole('button', { name: JOIN, exact: true })
    try {
      await expect
        .poll(async () => ((await join.count()) > 0 ? 'asks' : (await dialog.count()) === 0 ? 'closed' : 'waiting'), { timeout: STEP_LIMIT_MS })
        .not.toBe('waiting')
    } catch {
      return
    }
    if ((await join.count()) > 0) await join.click()
  }

  private getRows(panel: Locator): Locator {
    return panel.getByRole('list', { name: LIST }).getByRole('listitem')
  }

  async readRows(): Promise<DeviceRow[]> {
    const rows = this.getRows(await this.showPage())
    await expect(rows.first()).toBeVisible({ timeout: STEP_LIMIT_MS })
    const all = await rows.all()
    return Promise.all(
      all.map(async (row) => {
        const text = await row.innerText()
        return { fingerprint: readFingerprint(text), isThis: text.includes(THIS_DEVICE), isRevoked: text.includes(REVOKED) }
      }),
    )
  }

  /** Expects the list to hold this many devices that are not revoked. */
  async expectActiveDevices(count: number): Promise<void> {
    await expect
      .poll(async () => (await this.readRows()).filter((row) => !row.isRevoked).length, { timeout: STEP_LIMIT_MS })
      .toBe(count)
  }

  /** Expects the device list to show a device with this fingerprint that is not this one. */
  async expectOtherDevice(fingerprint: string): Promise<void> {
    await expect
      .poll(async () => (await this.readRows()).some((row) => row.fingerprint === fingerprint && !row.isThis), { timeout: STEP_LIMIT_MS })
      .toBe(true)
  }

  /** The fingerprint of this device in its own list. */
  async readThisFingerprint(): Promise<string> {
    const own = (await this.readRows()).find((row) => row.isThis)
    if (own === undefined) throw new Error('the list of devices has no row that says This device')
    return own.fingerprint
  }

  private async getRowOf(fingerprint: string): Promise<Locator> {
    const rows = this.getRows(await this.showPage())
    return rows.filter({ hasText: fingerprint })
  }

  /** Revokes the device with this fingerprint and confirms the question the page asks. */
  async revoke(fingerprint: string): Promise<void> {
    const row = await this.getRowOf(fingerprint)
    await row.click({ button: 'right' })
    await this.scenario.page.getByRole('menuitem', { name: REVOKE, exact: true }).click()
    await this.scenario.page.getByRole('alertdialog').getByRole('button', { name: REVOKE, exact: true }).click()
  }

  async expectNoRevokeOnThisDevice(): Promise<void> {
    const rows = this.getRows(await this.showPage())
    const own = rows.filter({ hasText: THIS_DEVICE })
    await expect(own).toHaveCount(1)
    await own.click({ button: 'right' })
    await expect(this.scenario.page.getByRole('menuitem', { name: REVOKE })).toHaveCount(0)
  }

  /** Expects a warning of the page, in the dialog or beside it. */
  async expectWarning(text: string): Promise<void> {
    await expect(this.scenario.page.getByRole('alert').filter({ hasText: text }).first()).toBeVisible({ timeout: STEP_LIMIT_MS })
  }

  /** Expects the page of a revoked device: the notice and the way out. */
  async expectRevoked(): Promise<void> {
    await this.closeDialog()
    const panel = await this.settings.showPage(PAGE)
    await expect(panel.getByRole('button', { name: DELETE_REVOKED, exact: true })).toBeVisible({ timeout: STEP_LIMIT_MS })
  }

  async deleteRevoked(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: DELETE_REVOKED, exact: true }).click()
    await this.scenario.page.getByRole('alertdialog').getByRole('button', { name: 'Delete', exact: true }).click()
  }

  async isRevoked(): Promise<boolean> {
    const panel = await this.showPage()
    return (await panel.getByRole('button', { name: DELETE_REVOKED, exact: true }).count()) > 0
  }
}

