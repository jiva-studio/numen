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
import { getShortFingerprint, ShareDriver } from './share.js'
import type { SyncDriver } from './sync.js'
import { SyncVaultsDriver } from './sync-vaults.js'

const SYNC_PLUGIN = 'md.numen.tools.sync'
const PLUGIN_NAME = 'Numen Sync'
const SHARES = 'Shared folders'
const ACCEPT = 'Accept a shared folder...'
const LINK = 'Invitation link'
/** The code the dialog shows while it waits for the owner: six digits in two groups. */
const CODE = /^\d{3} \d{3}$/
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
  readonly application: ApplicationDriver
  lastError: string | undefined
}

interface ReaderParts {
  readonly first: Scenario
  readonly viewport: ViewportSize | null
  readonly sync: SyncDriver
}

export class SharedReaderDriver {
  private readonly left = new Map<string, string>()
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
      vaults: new SyncVaultsDriver(scenario, settings, application),
      share: new ShareDriver(scenario, new FileTreeDriver(scenario), application, settings),
      application,
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

  private async openAccept(reader: Reader): Promise<Locator> {
    await reader.application.runCommand(ACCEPT)
    const dialog = reader.scenario.page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    return dialog
  }

  /**
   * Pastes the link, compares the owner the dialog shows with the identity of the person who shared (the two people
   * compare it by voice), asks for the folder and returns the code the dialog shows while it waits for the owner.
   */
  async ask(name: ReaderName, link: string, ownerFingerprint: string): Promise<string> {
    const dialog = await this.openAccept(this.require(name))
    await dialog.getByRole('textbox', { name: LINK }).fill(link)
    await expect(dialog.getByText(getShortFingerprint(ownerFingerprint), { exact: false })).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
    await dialog.getByRole('button', { name: 'Accept', exact: true }).click()
    const code = dialog.getByText(CODE)
    await expect(code).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
    return (await code.innerText()).trim()
  }

  /** Expects the open dialog to tell that the folder arrived, and closes it. */
  async expectReceived(name: ReaderName): Promise<void> {
    const { page } = this.require(name).scenario
    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('The folder is shared with you')).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
    await dialog.getByRole('button', { name: 'Done', exact: true }).click()
    await expect(dialog).toHaveCount(0)
  }

  /** Pastes a link and presses Accept; the dialog shows why when the link does not work. */
  async tryAccept(name: ReaderName, link: string): Promise<void> {
    const dialog = await this.openAccept(this.require(name))
    await dialog.getByRole('textbox', { name: LINK }).fill(link)
    const accept = dialog.getByRole('button', { name: 'Accept', exact: true })
    await expect(accept.or(dialog.getByRole('alert'))).toBeVisible({ timeout: ARRIVAL_LIMIT_MS })
    if (await accept.isEnabled()) await accept.click()
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

  /** Leaves the shared folder from the row menu of the page "Shared folders" and remembers where its folder was. */
  async leaveShare(name: ReaderName, vault: string): Promise<void> {
    const reader = this.require(name)
    const folder = await this.findVaultFolder(reader, vault)
    if (folder !== undefined) this.left.set(`${name}/${vault}`, folder)
    const panel = await this.showShares(reader)
    await panel.getByRole('button', { name: `Actions for ${vault}` }).click()
    await reader.scenario.page.getByRole('menuitem', { name: 'Leave', exact: true }).click()
    await reader.scenario.page.getByRole('alertdialog').getByRole('button', { name: 'Leave', exact: true }).click()
  }

  expectFolderKept(name: ReaderName, vault: string, filePath: string, text: string): void {
    const folder = this.left.get(`${name}/${vault}`)
    expect(folder, `the ${name} left no shared vault "${vault}"`).toBeDefined()
    expect(readNote(folder as string, filePath)).toBe(text)
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
  async shareVault(name: ReaderName, vault: string): Promise<void> {
    const reader = this.require(name)
    const found = (await listVaults(reader.scenario.page.request)).find((one) => one.name === vault)
    if (found === undefined) throw new Error(`the ${name} has no vault "${vault}"`)
    await callRPC(reader.scenario.page.request, 'VaultsService/OpenVault', { id: found.id })
    const started = await reader.share.start('')
    reader.lastError = 'refusal' in started ? started.refusal : undefined
    await reader.share.close()
  }

  async connectVault(name: ReaderName, vault: string): Promise<void> {
    const reader = this.require(name)
    reader.lastError = undefined
    await reader.vaults.connectNew(vault)
  }

  /** Expects the words in the answer of the core, in the open dialog, on the page "Shared folders" of the reader, or in the card of its subscription; with a vault, the reader must have that vault. */
  async expectTold(name: ReaderName, text: string, vault?: string): Promise<void> {
    const reader = this.require(name)
    const { page } = reader.scenario
    const isShown = (panel: Locator): Promise<boolean> => panel.getByText(text).first().isVisible()
    await expect
      .poll(
        async () => {
          if (vault !== undefined && (await this.findVaultFolder(reader, vault)) === undefined) return false
          if (reader.lastError?.includes(text) === true) return true
          const dialog = page.getByRole('dialog')
          if ((await dialog.count()) > 0) return isShown(dialog)
          if (await isShown(await this.showShares(reader))) return true
          return reader.subscription.isShown(text)
        },
        { message: `the ${name} is told "${text}"`, timeout: ARRIVAL_LIMIT_MS },
      )
      .toBe(true)
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
