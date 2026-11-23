/**
 * The computers of the person: the first is the application the scenario starts, the second is another application of
 * its own, with a home folder and a vault of its own, that reaches the same sync server.
 */
import { expect, type ViewportSize } from 'playwright/test'
import { ApplicationDriver } from './application.js'
import { DevicesDriver } from './devices.js'
import { IdentityDriver } from './identity.js'
import { makeWrongCode } from './link.js'
import { Scenario } from './scenario.js'
import { SettingsDriver } from './settings.js'
import type { SyncDriver } from './sync.js'
import { SyncVaultsDriver } from './sync-vaults.js'

/** The vault of the second computer: a scenario names it, and the computer has no other. */
const SECOND_VAULT = 'Copy'
/** A note arrives within a few passes of the plugin on both computers. */
const ARRIVAL_LIMIT_MS = 30_000
const ADDRESS_PATTERN = /^(?:the|its) vault "([^"]+)"$/

export type ComputerName = 'first' | 'second'

export interface Holder {
  readonly computer: ComputerName
  readonly vault: string | undefined
}

export const parseComputer = (text: string): ComputerName => {
  if (text === 'the person') return 'first'
  if (text === 'the second computer') return 'second'
  throw new Error(`"${text}" does not name a computer: use the person or the second computer`)
}

/** The vault in "the vault "Name"" or "its vault "Name"". */
export const parseOwnedVault = (text: string): string => {
  const match = ADDRESS_PATTERN.exec(text)
  if (match?.[1] === undefined) throw new Error(`"${text}" does not name a vault: write the vault "Name" or its vault "Name"`)
  return match[1]
}

/** Who has a note: the second computer, or the vault of the person. */
export const parseHolder = (text: string): Holder => {
  if (text === 'the second computer') return { computer: 'second', vault: undefined }
  return { computer: 'first', vault: parseOwnedVault(text) }
}

/** The vault after "in", or no vault where the step names none. */
export const parseInside = (text: string): string | undefined => {
  const trimmed = text.trim()
  return trimmed === '' ? undefined : parseOwnedVault(trimmed.replace(/^in /, ''))
}

/** The path of a note a scenario names by its title. */
export const getNotePath = (title: string): string => `${title}.md`

interface SecondComputer {
  readonly scenario: Scenario
  readonly identity: IdentityDriver
  readonly vaults: SyncVaultsDriver
  readonly devices: DevicesDriver
}

interface ComputerParts {
  readonly first: Scenario
  readonly viewport: ViewportSize | null
  readonly identity: IdentityDriver
  readonly sync: SyncDriver
  readonly firstVaults: SyncVaultsDriver
  readonly firstDevices: DevicesDriver
}

export class ComputerDriver {
  private second: SecondComputer | undefined

  private readonly first: Scenario
  private readonly viewport: ViewportSize | null
  private readonly identity: IdentityDriver
  private readonly sync: SyncDriver
  private readonly firstVaults: SyncVaultsDriver
  private readonly firstDevices: DevicesDriver
  private shownQr: { readonly from: ComputerName; readonly text: string } | undefined
  private secondFingerprint: string | undefined

  constructor(parts: ComputerParts) {
    this.first = parts.first
    this.viewport = parts.viewport
    this.identity = parts.identity
    this.sync = parts.sync
    this.firstVaults = parts.firstVaults
    this.firstDevices = parts.firstDevices
  }

  private requireSecond(): SecondComputer {
    if (this.second === undefined) throw new Error('there is no second computer: begin with a second computer restores the identity from the recovery words')
    return this.second
  }

  private getScenario(computer: ComputerName): Scenario {
    return computer === 'first' ? this.first : this.requireSecond().scenario
  }

  private getVaults(computer: ComputerName): SyncVaultsDriver {
    return computer === 'first' ? this.firstVaults : this.requireSecond().vaults
  }

  private async ensureSecond(): Promise<SecondComputer> {
    if (this.second !== undefined) return this.second
    const scenario = new Scenario(this.first.page, this.viewport)
    scenario.createVault(SECOND_VAULT)
    this.sync.setPlugin(scenario)
    const application = new ApplicationDriver(scenario)
    const settings = new SettingsDriver(scenario, application)
    this.second = {
      scenario,
      identity: new IdentityDriver(scenario, settings),
      vaults: new SyncVaultsDriver(scenario, settings),
      devices: new DevicesDriver(scenario, settings),
    }
    await application.startApplication('editor')
    return this.second
  }

  /** Starts a second application over a vault of its own and restores the identity of the person in it from the recovery words. */
  async restoreIdentity(): Promise<void> {
    const words = this.identity.getWords()
    const second = await this.ensureSecond()
    await second.identity.restoreFromWords(words.join(' '))
    await second.identity.expectIdentity(true)
  }

  private getDevices(computer: ComputerName): DevicesDriver {
    return computer === 'first' ? this.firstDevices : this.requireSecond().devices
  }

  private requireQr(): string {
    if (this.shownQr === undefined) throw new Error('no QR code was shown: begin with a step that shows one')
    return this.shownQr.text
  }

  /** The device of the person that has the account shows a QR code. */
  async showQrToLink(): Promise<void> {
    this.shownQr = { from: 'first', text: await this.firstDevices.showQrToLink() }
  }

  /** A new second computer shows a QR code to join the account. */
  async showQrToJoin(): Promise<void> {
    const second = await this.ensureSecond()
    this.shownQr = { from: 'second', text: await second.devices.showQrToJoin() }
  }

  /** A computer reads the QR code that the other one shows. */
  async scan(computer: ComputerName): Promise<void> {
    const text = this.requireQr()
    if (this.shownQr?.from === computer) throw new Error('a computer cannot scan the QR code it shows itself')
    if (computer === 'first') await this.firstDevices.scanToLink(text)
    else await (await this.ensureSecond()).devices.scanToJoin(text)
  }

  async approve(): Promise<void> {
    await this.firstDevices.approve()
  }

  async decline(): Promise<void> {
    await this.firstDevices.decline()
  }

  /** The dialog of the person shows the name and the fingerprint the new computer shows for itself. */
  async expectPeerIsSecond(): Promise<void> {
    const own = await this.requireSecond().devices.readOwnDevice()
    const seen = await this.firstDevices.readPeer()
    expect(seen).toStrictEqual(own)
  }

  async expectAsksToJoin(): Promise<void> {
    await this.requireSecond().devices.expectAskedToJoin(this.identity.getAccountFingerprint(), false)
  }

  /** The second computer holds the account of the person: it joins where it asks to, and has a session of its own. */
  async expectSecondHasAccount(): Promise<void> {
    const second = this.requireSecond()
    await second.devices.expectAccount(this.identity.getAccountFingerprint())
    this.secondFingerprint = await second.devices.readThisFingerprint()
    await this.sync.expectSessionOfDevice(this.secondFingerprint)
  }

  async expectSecondHasNoIdentity(): Promise<void> {
    const second = this.requireSecond()
    await second.devices.closeDialog()
    await second.identity.expectIdentity(false)
  }

  async deleteSecondIdentity(): Promise<void> {
    const second = this.requireSecond()
    await second.devices.closeDialog()
    await second.identity.delete()
    this.secondFingerprint = undefined
  }

  async expectSecondCannotJoin(): Promise<void> {
    await this.requireSecond().devices.expectNoJoin()
  }

  async createSecondIdentity(): Promise<void> {
    await (await this.ensureSecond()).identity.create()
  }

  /** The second computer types the code that the device of the person shows, or a code it is given. */
  async enterCode(code?: string): Promise<void> {
    const second = this.requireSecond()
    await second.devices.enterCode(code ?? (await this.firstDevices.readConfirmationCode()))
  }

  async enterWrongCodes(times: number): Promise<void> {
    const wrong = makeWrongCode(await this.firstDevices.readConfirmationCode())
    for (let attempt = 0; attempt < times; attempt += 1) await this.requireSecond().devices.enterCode(wrong)
  }

  async expectActiveDevices(count: number): Promise<void> {
    await this.firstDevices.expectActiveDevices(count)
  }

  /** The second computer joins the account of the person by the QR code of the person; a revoked identity of it is deleted first. */
  async linkSecond(): Promise<void> {
    const second = await this.ensureSecond()
    if (await second.devices.isRevoked()) await this.deleteSecondIdentity()
    await this.showQrToLink()
    await this.scan('second')
    await this.approve()
    await this.expectSecondHasAccount()
  }

  async revokeSecond(): Promise<void> {
    if (this.secondFingerprint === undefined) throw new Error('the second computer has no device in the account: begin with a step that links it')
    await this.firstDevices.revoke(this.secondFingerprint)
  }

  async expectCannotRevokeThisDevice(): Promise<void> {
    await this.firstDevices.expectNoRevokeOnThisDevice()
  }

  async expectNoSessionOfSecond(): Promise<void> {
    if (this.secondFingerprint === undefined) throw new Error('the second computer has no device in the account: begin with a step that links it')
    await this.sync.expectNoSessionOfDevice(this.secondFingerprint)
  }

  async expectWarning(computer: ComputerName, text: string): Promise<void> {
    await this.getDevices(computer).expectWarning(text)
  }

  private resolveVault(computer: ComputerName, vault: string | undefined): string {
    const scenario = this.getScenario(computer)
    return scenario.getVault(vault ?? SECOND_VAULT)
  }

  writeNote(computer: ComputerName, title: string, text: string, vault: string): void {
    this.getScenario(computer).getWorld().writeNote(this.resolveVault(computer, vault), getNotePath(title), text)
  }

  writeFile(computer: ComputerName, filePath: string, vault: string): void {
    this.getScenario(computer).getWorld().writeText(this.resolveVault(computer, vault), filePath, 'bytes')
  }

  async expectNote(holder: Holder, title: string, text: string, vault: string | undefined): Promise<void> {
    const world = this.getScenario(holder.computer).getWorld()
    const name = this.resolveVault(holder.computer, holder.vault ?? vault)
    await expect
      .poll(() => world.readText(name, getNotePath(title))?.trimEnd(), { timeout: ARRIVAL_LIMIT_MS })
      .toBe(text)
  }

  async expectNoFile(computer: ComputerName, filePath: string, vault: string): Promise<void> {
    const world = this.getScenario(computer).getWorld()
    const name = this.resolveVault(computer, vault)
    await expect.poll(() => world.getEntryKind(name, filePath), { timeout: ARRIVAL_LIMIT_MS }).toBeUndefined()
  }

  async connectNew(computer: ComputerName, vault: string): Promise<void> {
    await this.getVaults(computer).connectNew(vault)
  }

  async connectExisting(computer: ComputerName, vault: string, synced: string): Promise<void> {
    await this.getVaults(computer).connectExisting(vault, synced)
  }

  async expectCanConnect(computer: ComputerName, synced: string): Promise<void> {
    const vault = this.getScenario(computer).getWorld().getVaults().at(0)
    if (vault === undefined) throw new Error('the computer has no vault')
    await this.getVaults(computer).expectCanConnectTo(vault, synced)
  }

  async expectConnected(computer: ComputerName, vault: string): Promise<void> {
    await this.getVaults(computer).expectConnected(vault)
  }

  async expectSubscription(computer: ComputerName, text: string): Promise<void> {
    const identity = computer === 'first' ? this.identity : this.requireSecond().identity
    await identity.expectSubscription(text)
  }

  /** Ends the second computer: its window, its application and its folders. */
  async remove(): Promise<void> {
    const second = this.second
    this.second = undefined
    if (second !== undefined) await second.scenario.removeWorld()
  }
}
