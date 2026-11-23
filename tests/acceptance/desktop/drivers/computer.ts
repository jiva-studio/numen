/**
 * The computers of the person: the first is the application the scenario starts, the second is another application of
 * its own, with a home folder and a vault of its own, that reaches the same sync server.
 */
import { expect, type ViewportSize } from 'playwright/test'
import { ApplicationDriver } from './application.js'
import { IdentityDriver } from './identity.js'
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

/** What the first computer keeps of the person's account and vaults. */
interface FirstComputer {
  readonly identity: IdentityDriver
  readonly sync: SyncDriver
  readonly vaults: SyncVaultsDriver
}

interface SecondComputer {
  readonly scenario: Scenario
  readonly identity: IdentityDriver
  readonly vaults: SyncVaultsDriver
}

export class ComputerDriver {
  private second: SecondComputer | undefined

  constructor(
    private readonly first: Scenario,
    private readonly viewport: ViewportSize | null,
    private readonly owner: FirstComputer,
  ) {}

  private requireSecond(): SecondComputer {
    if (this.second === undefined) throw new Error('there is no second computer: begin with a second computer restores the identity from the recovery words')
    return this.second
  }

  private getScenario(computer: ComputerName): Scenario {
    return computer === 'first' ? this.first : this.requireSecond().scenario
  }

  private getVaults(computer: ComputerName): SyncVaultsDriver {
    return computer === 'first' ? this.owner.vaults : this.requireSecond().vaults
  }

  /** Starts a second application over a vault of its own and restores the identity of the person in it from the recovery words. */
  async restoreIdentity(): Promise<void> {
    const words = this.owner.identity.getWords()
    const scenario = new Scenario(this.first.page, this.viewport)
    scenario.createVault(SECOND_VAULT)
    this.owner.sync.setPlugin(scenario)
    const application = new ApplicationDriver(scenario)
    const settings = new SettingsDriver(scenario, application)
    const identity = new IdentityDriver(scenario, settings)
    this.second = { scenario, identity, vaults: new SyncVaultsDriver(scenario, settings) }
    await application.startApplication('editor')
    await identity.restoreFromWords(words.join(' '))
    await identity.expectIdentity(true)
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
    const identity = computer === 'first' ? this.owner.identity : this.requireSecond().identity
    await identity.expectSubscription(text)
  }

  async expectPlan(computer: ComputerName, name: string, gigabytes: number): Promise<void> {
    const identity = computer === 'first' ? this.owner.identity : this.requireSecond().identity
    await identity.expectPlan(name, gigabytes)
  }

  async expectActive(computer: ComputerName): Promise<void> {
    const identity = computer === 'first' ? this.owner.identity : this.requireSecond().identity
    await identity.expectActive()
  }

  /** Ends the second computer: its window, its application and its folders. */
  async remove(): Promise<void> {
    const second = this.second
    this.second = undefined
    if (second !== undefined) await second.scenario.removeWorld()
  }
}
