/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import type { Browser, Page, ViewportSize } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { destroyNote, destroyTrashed, listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import { deleteVersion, listVersionIds } from './text-versions.js'
import { DEFAULT_VAULT, parseReference, pickByTitle, pickVault } from './rules.js'
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
  private browser: Browser | undefined
  private currentPage: Page

  constructor(
    firstPage: Page,
    private readonly viewport: ViewportSize | null,
  ) {
    this.currentPage = firstPage
  }

  get page(): Page {
    return this.currentPage
  }

  get hasWorld(): boolean {
    return this.world !== undefined
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
  async startWorld(): Promise<void> {
    const world = this.getWorld()
    const [first] = world.getVaults()
    if (first === undefined) throw new Error('the world has no vault: begin with the empty vault "Name"')
    if (this.server === undefined) {
      this.server = await Server.start(world.getFolder(first))
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
    this.world = undefined
  }

  get isOpen(): boolean {
    return this.hasWorkspace
  }

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    this.filesBefore = await this.listEntries()
    await openWorkspace(this.page)
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
