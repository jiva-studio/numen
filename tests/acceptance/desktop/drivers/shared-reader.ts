/**
 * The readers of a share: each is another application of its own, with a home folder, a vault and a new identity, that
 * reaches the same sync server. A reader needs no subscription.
 */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator, type ViewportSize } from 'playwright/test'
import { callRPC, listVaults } from '../../../e2e/desktop/tests/support/vault.js'
import { ApplicationDriver } from './application.js'
import { FileTreeDriver } from './file-tree.js'
import { IdentityDriver } from './identity.js'
import { SubscriptionDriver } from './subscription.js'
import { Scenario } from './scenario.js'
import { SettingsDriver } from './settings.js'
import { ShareDriver, type GivenInvitation } from './share.js'
import type { SyncDriver } from './sync.js'
import { SyncVaultsDriver } from './sync-vaults.js'

const SYNC_PLUGIN = 'md.numen.tools.sync'
const PLUGIN_NAME = 'Numen Sync'
const SHARES = 'Shares'
const YOUR_CODE = 'Your code'
const INVITATION = 'Invitation'
const RECEIVE = 'Receive'
/** The own vault of a reader: the application needs one to start. */
const HOME_VAULT = 'Reader'
/** The folder of a reader the shared vaults are made in, outside every vault. */
const RECEIVED_FOLDER = 'shared'
/** A share arrives within a few passes of the plugin on both computers. */
const ARRIVAL_LIMIT_MS = 30_000

export type ReaderName = 'reader' | 'second reader'

export const parseReader = (text: string): ReaderName => {
  if (text === 'the reader') return 'reader'
  if (text === 'the second reader') return 'second reader'
  throw new Error(`"${text}" does not name a reader: use the reader or the second reader`)
}

/** The path of a note a scenario names by its title. */
export const getSharedNotePath = (title: string): string => `${title}.md`

interface Reader {
  readonly scenario: Scenario
  readonly identity: IdentityDriver
  readonly subscription: SubscriptionDriver
  readonly settings: SettingsDriver
  readonly vaults: SyncVaultsDriver
  readonly share: ShareDriver
  lastError: string | undefined
}

interface ReaderParts {
  readonly first: Scenario
  readonly viewport: ViewportSize | null
  readonly sync: SyncDriver
}

export class SharedReaderDriver {
  private readonly readers = new Map<ReaderName, Reader>()

  constructor(private readonly parts: ReaderParts) {}

  private require(name: ReaderName): Reader {
    const reader = this.readers.get(name)
    if (reader === undefined) throw new Error(`there is no ${name}: begin with ${name === 'reader' ? 'a' : 'a second'} reader creates an identity`)
    return reader
  }

  /** Starts an application over a vault of its own and makes a new identity in it. */
  async createIdentity(name: ReaderName): Promise<void> {
    if (this.readers.has(name)) throw new Error(`the ${name} has an identity already`)
    const scenario = new Scenario(this.parts.first.page, this.parts.viewport)
    scenario.createVault(HOME_VAULT)
    scenario.setPluginSetting(SYNC_PLUGIN, 'receivedFolder', path.join(scenario.getWorld().root, RECEIVED_FOLDER))
    await this.parts.sync.setPlugin(scenario)
    const application = new ApplicationDriver(scenario)
    const settings = new SettingsDriver(scenario, application)
    const identity = new IdentityDriver(scenario, settings)
    const subscription = new SubscriptionDriver(scenario, settings)
    await application.startApplication('editor')
    await identity.create()
    this.readers.set(name, {
      scenario,
      identity,
      subscription,
      settings,
      vaults: new SyncVaultsDriver(scenario, settings),
      share: new ShareDriver(scenario, new FileTreeDriver(scenario), application),
      lastError: undefined,
    })
  }

  getFingerprint(name: ReaderName): string {
    return this.require(name).identity.getAccountFingerprint()
  }

  private async showShares(reader: Reader): Promise<Locator> {
    const panel = await reader.settings.showPage(PLUGIN_NAME)
    const section = panel.getByRole('tab', { name: SHARES, exact: true })
    if ((await section.count()) > 0) await section.click()
    return panel
  }

  /** The code the reader gives the owner, read from the page "Shares". */
  async getCode(name: ReaderName): Promise<string> {
    const reader = this.require(name)
    const field = (await this.showShares(reader)).getByRole('textbox', { name: YOUR_CODE })
    await expect(field).toHaveValue(/^numen-reader-1\./, { timeout: ARRIVAL_LIMIT_MS })
    return field.inputValue()
  }

  /** Pastes the invitation, checks that the page shows the fingerprint of the owner and receives the share. */
  async receive(name: ReaderName, invitation: GivenInvitation, ownerFingerprint: string): Promise<void> {
    const panel = await this.showShares(this.require(name))
    await panel.getByRole('textbox', { name: INVITATION }).fill(invitation.text)
    await expect(panel.getByText(ownerFingerprint, { exact: true })).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
    await panel.getByRole('button', { name: RECEIVE, exact: true }).click()
  }

  /** Pastes an invitation and presses Receive; the page shows why when it refuses. */
  async tryReceive(name: ReaderName, invitation: GivenInvitation): Promise<void> {
    const reader = this.require(name)
    const panel = await this.showShares(reader)
    await panel.getByRole('textbox', { name: INVITATION }).fill(invitation.text)
    await panel.getByRole('button', { name: RECEIVE, exact: true }).click()
  }

  private async findVaultFolder(reader: Reader, vault: string): Promise<string | undefined> {
    return (await listVaults(reader.scenario.page.request)).find((one) => one.name === vault)?.path
  }

  async expectNote(name: ReaderName, vault: string, title: string, text: string): Promise<void> {
    const reader = this.require(name)
    await expect
      .poll(
        async () => {
          const folder = await this.findVaultFolder(reader, vault)
          return folder === undefined ? undefined : readNote(folder, getSharedNotePath(title))
        },
        { timeout: ARRIVAL_LIMIT_MS },
      )
      .toBe(text)
  }

  async expectNoFile(name: ReaderName, vault: string, filePath: string): Promise<void> {
    const reader = this.require(name)
    await expect
      .poll(
        async () => {
          const folder = await this.findVaultFolder(reader, vault)
          return folder === undefined ? undefined : readNote(folder, filePath)
        },
        { timeout: ARRIVAL_LIMIT_MS },
      )
      .toBeUndefined()
  }

  async expectNoVault(name: ReaderName, vault: string): Promise<void> {
    const vaultNames = (await listVaults(this.require(name).scenario.page.request)).map((one) => one.name)
    expect(vaultNames).not.toContain(vault)
  }

  /** Makes a note in the vault through the window, as the editor does; the core refuses the write and the reader receives that refusal. */
  async createNote(name: ReaderName, vault: string, title: string): Promise<void> {
    const reader = this.require(name)
    const { request } = reader.scenario.page
    const found = (await listVaults(request)).find((one) => one.name === vault)
    if (found === undefined) throw new Error(`the ${name} has no vault "${vault}"`)
    reader.lastError = undefined
    try {
      await callRPC(request, 'VaultsService/OpenVault', { id: found.id })
      await callRPC(request, 'NoteService/CreateNote', { title })
    } catch (error) {
      reader.lastError = error instanceof Error ? error.message : String(error)
    }
  }

  /** Tries to share the shared vault with another reader; the dialog or the page tells why it cannot. */
  async shareVault(name: ReaderName, vault: string, other: ReaderName): Promise<void> {
    const reader = this.require(name)
    const found = (await listVaults(reader.scenario.page.request)).find((one) => one.name === vault)
    if (found === undefined) throw new Error(`the ${name} has no vault "${vault}"`)
    await callRPC(reader.scenario.page.request, 'VaultsService/OpenVault', { id: found.id })
    reader.lastError = await reader.share.tryAddReader('', await this.getCode(other), 'Other')
  }

  async connectVault(name: ReaderName, vault: string): Promise<void> {
    const reader = this.require(name)
    reader.lastError = undefined
    await reader.vaults.connectNew(vault)
  }

  /** Expects the words in the answer of the core, in the dialog, or on the page "Shares" of the reader. */
  async expectTold(name: ReaderName, text: string): Promise<void> {
    const reader = this.require(name)
    if (reader.lastError?.includes(text) === true) return
    const panel = await this.showShares(reader)
    await expect(panel.getByText(text).first()).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
  }

  async expectSubscription(name: ReaderName, text: string): Promise<void> {
    await this.require(name).subscription.expectSubscription(text)
  }

  async remove(): Promise<void> {
    const readers = [...this.readers.values()]
    this.readers.clear()
    for (const reader of readers) await reader.scenario.removeWorld()
  }
}

const readNote = (folder: string, filePath: string): string | undefined => {
  const target = path.join(folder, filePath)
  return fs.existsSync(target) ? fs.readFileSync(target, 'utf8').trimEnd() : undefined
}
