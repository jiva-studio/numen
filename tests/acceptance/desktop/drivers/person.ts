/**
 * The person at the window. Every method is one thing a person does or one outcome the product
 * holds, grouped by the level a step works at: the application, a tab, a note, the file tree, the search, the plex, the web. How it is done on
 * the screen (the tabs, the keys, the labels) stays in drivers/, so a step and a scenario say what
 * happens and never how it looks.
 */
import type { Page } from 'playwright/test'
import { ApplicationDriver } from './application.js'
import { FileTreeDriver } from './file-tree.js'
import { NoteDriver } from './notes.js'
import { PlexDriver } from './plex.js'
import { Scenario } from './scenario.js'
import { SearchDriver } from './search.js'
import { TabDriver } from './tabs.js'
import { WebDriver } from './web.js'

export class Person {
  readonly application: ApplicationDriver
  readonly tab: TabDriver
  readonly note: NoteDriver
  readonly fileTree: FileTreeDriver
  readonly search: SearchDriver
  readonly plex: PlexDriver
  readonly web: WebDriver
  private readonly scenario: Scenario

  constructor(page: Page) {
    this.scenario = new Scenario(page)
    this.application = new ApplicationDriver(this.scenario)
    this.tab = new TabDriver(this.scenario)
    this.note = new NoteDriver(this.scenario)
    this.fileTree = new FileTreeDriver(this.scenario)
    this.search = new SearchDriver(this.scenario)
    this.plex = new PlexDriver(this.scenario)
    this.web = new WebDriver()
  }

  async removeNotes(): Promise<void> {
    this.web.removePageTexts()
    await this.scenario.removeNotes()
  }
}
