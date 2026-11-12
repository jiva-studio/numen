/**
 * The person at the window. Every method is one thing a person does or one outcome the product
 * holds, grouped by the level a step works at: the application, a tab, a note, the file tree, the search, the plex, the web. How it is done on
 * the screen (the tabs, the keys, the labels) stays in drivers/, so a step and a scenario say what
 * happens and never how it looks.
 */
import type { Page, ViewportSize } from 'playwright/test'
import { AgentDriver } from './agent.js'
import { ApplicationDriver } from './application.js'
import { CommandDriver } from './command.js'
import { ConnectionsDriver } from './connections.js'
import { DeckDriver } from './deck.js'
import { parseEnvironment } from './environment.js'
import { FileTreeDriver } from './file-tree.js'
import { NoteDriver } from './notes.js'
import { PaletteDriver } from './palette.js'
import { PlexDriver } from './plex.js'
import { ReviewDriver } from './review.js'
import { PluginDriver } from './plugins.js'
import { ReaderDriver } from './reader.js'
import { Scenario } from './scenario.js'
import { ServiceDriver } from './services.js'
import { SearchDriver } from './search.js'
import { SettingsDriver } from './settings.js'
import { TabDriver } from './tabs.js'
import { TextDriver } from './text.js'
import { VaultDriver } from './vault.js'
import { WebDriver } from './web.js'
import { WindowDriver } from './window.js'

export class Person {
  readonly application: ApplicationDriver
  readonly tab: TabDriver
  readonly note: NoteDriver
  readonly fileTree: FileTreeDriver
  readonly search: SearchDriver
  readonly plex: PlexDriver
  readonly plugin: PluginDriver
  readonly web: WebDriver
  readonly text: TextDriver
  readonly review: ReviewDriver
  readonly vault: VaultDriver
  readonly command: CommandDriver
  readonly palette: PaletteDriver
  readonly window: WindowDriver
  readonly settings: SettingsDriver
  readonly agent: AgentDriver
  readonly services: ServiceDriver
  readonly connections: ConnectionsDriver
  readonly reader: ReaderDriver
  readonly deck: DeckDriver
  readonly scenario: Scenario

  constructor(page: Page, viewport: ViewportSize | null) {
    this.scenario = new Scenario(page, viewport)
    this.application = new ApplicationDriver(this.scenario)
    this.tab = new TabDriver(this.scenario)
    this.note = new NoteDriver(this.scenario)
    this.fileTree = new FileTreeDriver(this.scenario)
    this.search = new SearchDriver(this.scenario)
    this.plex = new PlexDriver(this.scenario)
    this.plugin = new PluginDriver(this.scenario)
    this.web = new WebDriver(this.scenario)
    this.window = new WindowDriver(this.scenario)
    this.text = new TextDriver(this.scenario, this.application, this.tab, this.window)
    this.review = new ReviewDriver(this.scenario)
    this.vault = new VaultDriver(this.scenario)
    this.palette = new PaletteDriver(this.scenario)
    this.command = new CommandDriver(this.scenario, {
      application: this.application,
      tab: this.tab,
      fileTree: this.fileTree,
      palette: this.palette,
    })
    this.settings = new SettingsDriver(this.scenario, this.application)
    this.agent = new AgentDriver(this.scenario)
    this.services = new ServiceDriver()
    this.connections = new ConnectionsDriver(this.scenario, this.settings, this.services)
    this.reader = new ReaderDriver(this.scenario, this.tab)
    this.deck = new DeckDriver(this.scenario, this.tab)
  }

  /** Sets the environment of the application; a release that comes out while the window is open is looked for again. */
  async setEnvironment(rows: readonly (readonly string[])[]): Promise<void> {
    const settings = parseEnvironment(rows)
    await this.scenario.setEnvironment(settings)
    if (this.scenario.isOpen && settings.some((one) => one.name === 'latest release')) await this.window.askFeedAgain()
  }

  readServerLog(): string | undefined {
    return this.scenario.readServerLog()
  }

  /** Leaves the vault as the scenario found it; a scenario with a world of its own removes the world and its server. */
  async removeNotes(): Promise<void> {
    await this.services.stopAll()
    if (this.scenario.hasWorld) return this.scenario.removeWorld()
    await this.plugin.enableDisabled()
    await this.scenario.removeVersions()
    this.web.removePageTexts()
    await this.review.takeBackAnswer()
    await this.scenario.removeNotes()
  }
}
