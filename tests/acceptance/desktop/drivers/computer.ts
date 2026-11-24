/**
 * The computers of the person: the first is the application the scenario starts, the second is another application of
 * its own, with a home folder and a vault of its own, that reaches the same sync server.
 */
import { expect, type ViewportSize } from 'playwright/test'
import { ApplicationDriver } from './application.js'
import { DevicesDriver } from './devices.js'
import { IdentityDriver } from './identity.js'
import { SubscriptionDriver } from './subscription.js'
import { makeWrongCode } from './link.js'
import { DEFAULT_VAULT } from './rules.js'
import { Scenario } from './scenario.js'
import { SettingsDriver } from './settings.js'
import { getIntervalSeconds, type SyncDriver } from './sync.js'
import type { World } from './world.js'
import { SyncVaultsDriver } from './sync-vaults.js'

/** The vault of the second computer: a scenario names it, and the computer has no other. */
const SECOND_VAULT = 'Copy'
/** A note arrives within a few passes of the plugin on both computers. */
const ARRIVAL_LIMIT_MS = 30_000
/** How long the plugin has to make a copy before a scenario may say there is none: a few passes. */
const QUIET_MS = 6_000
/** The vault of the computer that only reads the synced vault, to see what the server holds. */
const CHECK_VAULT = 'Check'
/** The server holds no note when a computer that joined waits this long and has none. */
const ABSENCE_WAIT_MS = 8_000
/** The plugin of the sync, by the name its settings page has. */
const SYNC_PLUGIN_NAME = 'Numen Sync'
/** The setting that says how often a computer asks the sync server for changes. */
const INTERVAL_SETTING = 'Sync interval'
/** Both computers hold a stream open to the server once their plugin runs. */
const STREAMS_OF_BOTH = 2
const ADDRESS_PATTERN = /^(?:the|its) vault "([^"]+)"$/

/** The folders of a computer and the name of one vault in them. */
export interface Place {
  readonly world: World
  readonly vault: string
}

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

const escapePattern = (text: string): string => text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** The texts of the conflict copies of a note: the files named "Title (conflict 2026-11-23).md", with a number after the date once that name is taken. */
export const findConflictCopies = (files: readonly { path: string; text: string }[], title: string): string[] => {
  const pattern = new RegExp(`^${escapePattern(title)} \\(conflict \\d{4}-\\d{2}-\\d{2}(?: \\d+)?\\)\\.md$`)
  return files.filter((file) => pattern.test(file.path)).map((file) => file.text.trimEnd())
}

/** What the first computer keeps of the person's account and vaults. */
interface FirstComputer {
  readonly identity: IdentityDriver
  readonly subscription: SubscriptionDriver
  readonly sync: SyncDriver
  readonly vaults: SyncVaultsDriver
  readonly settings: SettingsDriver
  readonly devices: DevicesDriver
}

interface SecondComputer {
  readonly scenario: Scenario
  readonly identity: IdentityDriver
  readonly subscription: SubscriptionDriver
  readonly vaults: SyncVaultsDriver
  readonly settings: SettingsDriver
  readonly devices: DevicesDriver
}

export class ComputerDriver {
  private second: SecondComputer | undefined
  private checker: SecondComputer | undefined
  private shownQr: { readonly from: ComputerName; readonly text: string } | undefined
  private secondFingerprint: string | undefined

  constructor(
    private readonly first: Scenario,
    private readonly viewport: ViewportSize | null,
    private readonly owner: FirstComputer,
  ) {}

  private requireSecond(): SecondComputer {
    if (this.second === undefined) throw new Error('there is no second computer: begin with a second computer restores the identity from the recovery words')
    return this.second
  }

  getScenario(computer: ComputerName): Scenario {
    return computer === 'first' ? this.first : this.requireSecond().scenario
  }

  private getVaults(computer: ComputerName): SyncVaultsDriver {
    return computer === 'first' ? this.owner.vaults : this.requireSecond().vaults
  }

  private async ensureSecond(): Promise<SecondComputer> {
    this.second ??= await this.startComputer(SECOND_VAULT)
    return this.second
  }

  private async startComputer(vault: string): Promise<SecondComputer> {
    const scenario = new Scenario(this.first.page, this.viewport)
    scenario.createVault(vault)
    await this.owner.sync.setPlugin(scenario)
    const application = new ApplicationDriver(scenario)
    const settings = new SettingsDriver(scenario, application)
    const started = {
      scenario,
      identity: new IdentityDriver(scenario, settings),
      subscription: new SubscriptionDriver(scenario, settings),
      vaults: new SyncVaultsDriver(scenario, settings),
      devices: new DevicesDriver(scenario, settings),
      settings,
    }
    await application.startApplication('editor')
    return started
  }

  private async restoreWords(computer: SecondComputer): Promise<void> {
    const words = this.owner.identity.getWords()
    await computer.identity.restoreFromWords(words.join(' '))
    await computer.identity.expectIdentity(true)
  }

  /** Starts a second application over a vault of its own and restores the identity of the person in it from the recovery words. */
  async restoreIdentity(): Promise<void> {
    await this.restoreWords(await this.ensureSecond())
  }

  /** Reads the synced vault the way a person does: a computer that connects its own vault to it, and the note it receives. */
  async expectHeld(synced: string, title: string, text: string | undefined): Promise<void> {
    this.checker ??= await this.startChecker(synced)
    const world = this.checker.scenario.getWorld()
    const path = getNotePath(title)
    if (text !== undefined) {
      await expect.poll(() => world.readText(CHECK_VAULT, path)?.trimEnd(), { timeout: ARRIVAL_LIMIT_MS }).toBe(text)
      return
    }
    await new Promise((resolve) => setTimeout(resolve, ABSENCE_WAIT_MS))
    expect(world.getEntryKind(CHECK_VAULT, path)).toBeUndefined()
  }

  private async startChecker(synced: string): Promise<SecondComputer> {
    const started = await this.startComputer(CHECK_VAULT)
    await this.restoreWords(started)
    await started.vaults.connectExisting(CHECK_VAULT, synced)
    return started
  }

  private getDevices(computer: ComputerName): DevicesDriver {
    return computer === 'first' ? this.owner.devices : this.requireSecond().devices
  }

  private requireQr(): string {
    if (this.shownQr === undefined) throw new Error('no QR code was shown: begin with a step that shows one')
    return this.shownQr.text
  }

  /** The device of the person that has the account shows a QR code. */
  async showQrToLink(): Promise<void> {
    this.shownQr = { from: 'first', text: await this.owner.devices.showQrToLink() }
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
    if (computer === 'first') await this.owner.devices.scanToLink(text)
    else await (await this.ensureSecond()).devices.scanToJoin(text)
  }

  async approve(): Promise<void> {
    await this.owner.devices.approve()
  }

  async decline(): Promise<void> {
    await this.owner.devices.decline()
  }

  /** The dialog of the person shows the name and the fingerprint the new computer shows for itself. */
  async expectPeerIsSecond(): Promise<void> {
    const own = await this.requireSecond().devices.readOwnDevice()
    const seen = await this.owner.devices.readPeer()
    expect(seen).toStrictEqual(own)
  }

  async expectAsksToJoin(): Promise<void> {
    await this.requireSecond().devices.expectAskedToJoin(this.owner.identity.getAccountFingerprint(), false)
  }

  /** The second computer holds the account of the person: it joins where it asks to, and has a session of its own. */
  async expectSecondHasAccount(): Promise<void> {
    const second = this.requireSecond()
    await second.devices.expectAccount(this.owner.identity.getAccountFingerprint())
    this.secondFingerprint = await second.devices.readThisFingerprint()
    await this.owner.sync.expectSessionOfDevice(this.secondFingerprint)
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
    await second.devices.enterCode(code ?? (await this.owner.devices.readConfirmationCode()))
  }

  async enterWrongCodes(times: number): Promise<void> {
    const wrong = makeWrongCode(await this.owner.devices.readConfirmationCode())
    for (let attempt = 0; attempt < times; attempt += 1) await this.requireSecond().devices.enterCode(wrong)
  }

  async expectActiveDevices(count: number): Promise<void> {
    await this.owner.devices.expectActiveDevices(count)
  }

  /** The second computer joins the account of the person by the QR code of the person; a revoked identity of it is deleted first. */
  async linkSecond(): Promise<void> {
    const second = await this.ensureSecond()
    if (await second.devices.isRevoked()) await this.deleteSecondIdentity()
    await this.showQrToLink()
    await this.scan('second')
    await this.approve()
    await this.enterCode()
    await this.expectSecondHasAccount()
  }

  async revokeSecond(): Promise<void> {
    if (this.secondFingerprint === undefined) throw new Error('the second computer has no device in the account: begin with a step that links it')
    await this.owner.devices.revoke(this.secondFingerprint)
  }

  async expectCannotRevokeThisDevice(): Promise<void> {
    await this.owner.devices.expectNoRevokeOnThisDevice()
  }

  async expectNoSessionOfSecond(): Promise<void> {
    if (this.secondFingerprint === undefined) throw new Error('the second computer has no device in the account: begin with a step that links it')
    await this.owner.sync.expectNoSessionOfDevice(this.secondFingerprint)
  }

  async expectWarning(computer: ComputerName, text: string): Promise<void> {
    await this.getDevices(computer).expectWarning(text)
  }

  private resolveVault(computer: ComputerName, vault: string | undefined): string {
    const scenario = this.getScenario(computer)
    return scenario.getVault(vault ?? SECOND_VAULT)
  }

  /** The folders of a computer and the name of the vault a step names, or the only vault of the second computer. */
  getPlace(computer: ComputerName, vault: string | undefined): Place {
    const named = vault ?? (computer === 'first' ? DEFAULT_VAULT : undefined)
    return { world: this.getScenario(computer).getWorld(), vault: this.resolveVault(computer, named) }
  }

  writeNote(computer: ComputerName, title: string, text: string, vault: string): void {
    this.getScenario(computer).getWorld().writeNote(this.resolveVault(computer, vault), getNotePath(title), text)
  }

  writeFile(computer: ComputerName, filePath: string, vault: string): void {
    this.getScenario(computer).getWorld().writeText(this.resolveVault(computer, vault), filePath, 'bytes')
  }

  async expectNote(holder: Holder, title: string, text: string, vault: string | undefined): Promise<void> {
    await this.expectNoteBy({ ...holder, vault: holder.vault ?? vault }, title, text, ARRIVAL_LIMIT_MS)
  }

  private async expectNoteBy(holder: Holder, title: string, text: string, timeout: number): Promise<void> {
    const world = this.getScenario(holder.computer).getWorld()
    const name = this.resolveVault(holder.computer, holder.vault)
    await expect.poll(() => world.readText(name, getNotePath(title))?.trimEnd(), { timeout }).toBe(text)
  }

  /** Asks every computer for changes only after the interval the text names. */
  async setInterval(text: string): Promise<void> {
    const seconds = String(getIntervalSeconds(text))
    await this.owner.settings.set(INTERVAL_SETTING, seconds, SYNC_PLUGIN_NAME)
    await this.requireSecond().settings.set(INTERVAL_SETTING, seconds, SYNC_PLUGIN_NAME)
    await this.owner.sync.expectStreams(STREAMS_OF_BOTH)
  }

  /** Takes the second computer offline for a moment: the server ends its open connections and waits until none is open. */
  async dropConnection(): Promise<void> {
    await this.owner.sync.dropStreams()
  }

  /** Expects the note within the seconds the scenario names, with the settings the plugin has now. */
  async expectNoteWithin(holder: Holder, note: { title: string; text: string; seconds: number }): Promise<void> {
    await this.expectNoteBy(holder, note.title, note.text, note.seconds * 1000)
  }

  async expectConflictCopy(holder: Holder, title: string, text: string, vault: string | undefined): Promise<void> {
    const world = this.getScenario(holder.computer).getWorld()
    const name = this.resolveVault(holder.computer, holder.vault ?? vault)
    await expect
      .poll(() => findConflictCopies(world.listNoteFiles(name), title), { timeout: ARRIVAL_LIMIT_MS })
      .toContain(text)
  }

  /** The copies a conflict makes arrive within a few passes, so the check waits for the passes to end before it looks. */
  async expectNoConflictCopy(holder: Holder, title: string, vault: string | undefined): Promise<void> {
    const world = this.getScenario(holder.computer).getWorld()
    const name = this.resolveVault(holder.computer, holder.vault ?? vault)
    await new Promise((resolve) => setTimeout(resolve, QUIET_MS))
    expect(findConflictCopies(world.listNoteFiles(name), title)).toEqual([])
  }

  /** Makes the other computer reach the sync server only after this one sent its changes. */
  connectToServer(computer: ComputerName): void {
    this.owner.sync.holdOthers(this.getScenario(computer))
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
    const subscription = computer === 'first' ? this.owner.subscription : this.requireSecond().subscription
    await subscription.expectSubscription(text)
  }

  async expectPlan(computer: ComputerName, name: string, gigabytes: number): Promise<void> {
    const subscription = computer === 'first' ? this.owner.subscription : this.requireSecond().subscription
    await subscription.expectPlan(name, gigabytes)
  }

  async expectActive(computer: ComputerName): Promise<void> {
    const subscription = computer === 'first' ? this.owner.subscription : this.requireSecond().subscription
    await subscription.expectActive()
  }

  removeNote(computer: ComputerName, title: string, vault: string): void {
    this.getScenario(computer).getWorld().removeFile(this.resolveVault(computer, vault), getNotePath(title))
  }

  async expectNoNote(holder: Holder, title: string, vault: string | undefined): Promise<void> {
    const world = this.getScenario(holder.computer).getWorld()
    const name = this.resolveVault(holder.computer, holder.vault ?? vault)
    await expect.poll(() => world.getEntryKind(name, getNotePath(title)), { timeout: ARRIVAL_LIMIT_MS }).toBeUndefined()
  }

  /** Makes the computer unable to reach the sync server, or able again. */
  setOnline(computer: ComputerName, isOnline: boolean): void {
    this.sync.setReachable(this.getScenario(computer), isOnline)
  }

  async answerConfirmation(computer: ComputerName, answer: 'Connect' | 'Cancel'): Promise<void> {
    await this.getVaults(computer).answerConfirmation(answer)
  }

  async expectAskedToConfirm(computer: ComputerName, text: string): Promise<void> {
    await this.getVaults(computer).expectAskedToConfirm(text)
  }

  async expectNotAsked(computer: ComputerName): Promise<void> {
    await this.getVaults(computer).expectNotAsked()
  }

  async expectNotConnected(computer: ComputerName, vault: string): Promise<void> {
    await this.getVaults(computer).expectNotConnected(vault)
  }

  /** Ends the second computer: its window, its application and its folders. */
  async remove(): Promise<void> {
    const second = this.second
    const checker = this.checker
    this.second = undefined
    this.checker = undefined
    if (second !== undefined) await second.scenario.removeWorld()
    if (checker !== undefined) await checker.scenario.removeWorld()
  }
}
