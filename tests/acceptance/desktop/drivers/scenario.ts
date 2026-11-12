/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import type { Browser, Page, ViewportSize } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { destroyNote, destroyTrashed, listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { getGraph, openWorkspace, waitForWindow } from '../../../e2e/desktop/tests/support/workspace.js'
import { expect } from 'playwright/test'
import { deleteVersion, listVersionIds } from './text-versions.js'
import {
  assertWorldApplication,
  DEFAULT_VAULT,
  getNoteFile,
  parseReference,
  pickByTitle,
  pickVault,
  type Application,
} from './rules.js'
import type { EnvironmentSetting } from './environment.js'
import { Palette } from './components/Palette.js'
import { Server } from './server.js'
import { World } from './world.js'

export class Scenario {
  private readonly paths = new Map<string, string>()
  private readonly titles = new Set<string>()
  private hasWorkspace = false
  private filesBefore: readonly string[] = []
  private readonly versionsBefore = new Map<string, readonly string[]>()
  private vaultName: string | undefined
  private world: World | undefined
  private server: Server | undefined
  private application: Application | undefined
  private isRestarted = false
  private browser: Browser | undefined
  private currentPage: Page
  private environment: EnvironmentSetting[] = []
  private actionCount = 0

  constructor(
    firstPage: Page,
    private readonly viewport: ViewportSize | null,
  ) {
    this.currentPage = firstPage
  }

  get page(): Page {
    return this.currentPage
  }

  /** How many commands the person ran: a step that declares a state and one that checks it share their words. */
  get actions(): number {
    return this.actionCount
  }

  noteAction(): void {
    this.actionCount += 1
  }

  /** Sets the environment of the application now, or from the start of it when it has not started. */
  async setEnvironment(settings: readonly EnvironmentSetting[]): Promise<void> {
    this.environment = [...this.environment.filter((old) => !settings.some((one) => one.name === old.name)), ...settings]
    await this.server?.setEnvironment(settings)
  }

  getServer(): Server {
    if (this.server === undefined) throw new Error('the application has not started: begin with the application "editor" starts')
    return this.server
  }

  get hasWorld(): boolean {
    return this.world !== undefined
  }

  get hasServer(): boolean {
    return this.server !== undefined
  }

  /** Gives the scenario a world of its own, if it has none, and puts an empty vault of that name in it. */
  createVault(name: string): void {
    this.world ??= World.create()
    this.world.addVault(name)
  }

  getWorld(): World {
    if (this.world === undefined) throw new Error('the scenario has no world: begin with the empty vault "Name"')
    return this.world
  }

  /** The vault of the world a step names; a step of a scenario with no world can only name its vault. */
  getVault(name: string): string {
    return pickVault(this.getWorld().getVaults(), name)
  }

  /** Starts the server of the world on its first vault, or starts it again over the same state, and opens a window on it. */
  async startWorld(application: Application): Promise<void> {
    const world = this.getWorld()
    const vaults = world.getVaults()
    if (vaults.length === 0) throw new Error('the world has no vault: begin with the empty vault "Name"')
    assertWorldApplication(application, vaults, this.application)
    this.isRestarted = this.server !== undefined
    if (this.server === undefined) {
      this.server = await Server.start({
        application,
        vault: world.getFolder(vaults[0]),
        readPageText: (address) => world.readPageText(address),
        environment: this.environment,
      })
      this.application = application
    } else {
      await this.server.restart()
    }
    await this.browser?.close()
    const browserType = this.currentPage.context().browser()?.browserType()
    if (browserType === undefined) throw new Error('the page has no browser to launch another from')
    const args = browserType.name() === 'chromium' ? [`--ignore-certificate-errors-spki-list=${this.server.getPublicKeyHash()}`] : []
    this.browser = await browserType.launch({ args })
    const context = await this.browser.newContext({
      baseURL: this.server.getBaseURL(),
      ignoreHTTPSErrors: true,
      viewport: this.viewport,
    })
    // A link without a copy embeds the player of the site, which a run offline does not reach.
    await context.route(/youtube-nocookie\.com/, (route) => route.abort())
    this.currentPage = await context.newPage()
    this.hasWorkspace = false
  }

  readServerLog(): string | undefined {
    return this.server?.readLog()
  }

  /** Ends the window, the server and the world, in this order, and removes every folder they made. */
  async removeWorld(): Promise<void> {
    try {
      await this.browser?.close()
    } finally {
      try {
        await this.server?.stop()
      } finally {
        this.world?.remove()
      }
    }
    this.browser = undefined
    this.server = undefined
    this.application = undefined
    this.world = undefined
  }

  get isOpen(): boolean {
    return this.hasWorkspace
  }

  /** Closes the palette or the search when it is open, as it covers the window and takes the clicks. */
  async closePalette(): Promise<void> {
    await new Palette(this).close()
  }

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    if (this.world !== undefined) return this.openWorldWindow(this.world)
    this.filesBefore = await this.listEntries()
    await openWorkspace(this.page)
    this.hasWorkspace = true
  }

  private async openWorldWindow(world: World): Promise<void> {
    if (this.server === undefined) await this.startWorld('editor')
    assertWorldApplication('editor', world.getVaults(), this.application)
    await this.page.goto('/')
    await waitForWindow(this.page)
    // Only the first start opens the plex of a note: a restart opens the tabs the person left.
    if (!this.isRestarted && world.hasNotes()) await expect(getGraph(this.page)).toBeVisible()
    this.hasWorkspace = true
  }

  /** The paths of the files and folders of the vault below `folder`; the contents of a folder come before the folder. */
  async listEntries(folder = ''): Promise<string[]> {
    const entries = await listFiles(this.page.request, folder)
    const nested = await Promise.all(entries.filter((one) => one.isFolder).map((one) => this.listEntries(one.path)))
    return [...nested.flat(), ...entries.map((one) => one.path)]
  }

  /** Notes the text versions a file has when the person opens it, so the scenario can delete the ones it makes. */
  async watchVersions(filePath: string): Promise<void> {
    if (this.versionsBefore.has(filePath)) return
    this.versionsBefore.set(filePath, await listVersionIds(this.page.request, filePath).catch(() => []))
  }

  async claimTitle(title: string): Promise<void> {
    this.titles.add(title)
    await destroyNote(this.page.request, `notes/${title}.md`).catch(() => undefined)
  }

  rememberNote(title: string, notePath: string): void {
    this.paths.set(title, notePath)
  }

  async getPath(reference: string, name: string): Promise<string> {
    if (parseReference(reference, name).reference === 'at') return name
    const savedPath = this.paths.get(name)
    if (savedPath) return savedPath
    return pickByTitle(name, [...(await readSavedStates(this.page.request)).keys()])
  }

  /** The path of the note a step wrote or the person made: the file the product made, or the file in the world. */
  getNotePath(title: string): string {
    const remembered = this.paths.get(title)
    if (remembered !== undefined) return remembered
    if (this.world === undefined) throw new Error(`no note is titled "${title}": begin with the vault has the following: a row with the path of the note and its text`)
    return getNoteFile(title)
  }

  /** The vault a step names when the scenario has no world: the shared vault, named by the run or not at all. */
  checkSharedVault(name: string): void {
    if (name !== DEFAULT_VAULT) this.setVault(name)
  }

  setVault(name: string): void {
    if (name === DEFAULT_VAULT) throw new Error('name the vault: the scenario has no world, so its vault has the name the run gives')
    this.vaultName ??= name
    if (this.vaultName !== name) {
      throw new Error(`the run has one vault, named "${this.vaultName}" here, not "${name}"`)
    }
  }

  /** Deletes the text versions the scenario made over the files it opened: a file of the vault outlives the scenario. */
  async removeVersions(): Promise<void> {
    const { request } = this.page
    for (const [filePath, before] of this.versionsBefore) {
      const now = await listVersionIds(request, filePath).catch(() => [])
      for (const id of now.filter((one) => !before.includes(one))) await deleteVersion(request, filePath, id).catch(() => undefined)
    }
    this.versionsBefore.clear()
  }

  async removeNotes(): Promise<void> {
    const made = this.hasWorkspace
      ? (await this.listEntries()).filter((one) => !this.filesBefore.includes(one))
      : []
    const paths = [...this.paths.values(), ...[...this.titles].map((title) => `notes/${title}.md`), ...made]
    for (const notePath of paths) await destroyNote(this.page.request, notePath).catch(() => undefined)
    for (const title of this.titles) destroyTrashed(`${title}.md`)
  }
}
