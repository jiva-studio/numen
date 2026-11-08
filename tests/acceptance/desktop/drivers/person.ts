/**
 * The person at the window. Every method is one thing a person does or one outcome the product
 * holds, grouped by the level a step works at: the application, a tab, a note, the search, the web. How it is done on
 * the screen (the tabs, the keys, the labels) stays in drivers/, so a step and a scenario say what
 * happens and never how it looks.
 */
import type { Page } from 'playwright/test'
import { ApplicationDriver } from './application.js'
import { NoteDriver } from './notes.js'
import { Scenario } from './scenario.js'
import { SearchDriver } from './search.js'
import { TabDriver } from './tabs.js'
import { WebDriver } from './web.js'

export class Person {
  readonly application: ApplicationDriver
  readonly tab: TabDriver
  readonly note: NoteDriver
  readonly search: SearchDriver
  readonly web: WebDriver
  private readonly scenario: Scenario

  constructor(page: Page) {
    this.scenario = new Scenario(page)
    this.application = new ApplicationDriver(this.scenario)
    this.tab = new TabDriver(this.scenario)
    this.note = new NoteDriver(this.scenario)
    this.search = new SearchDriver(this.scenario)
    this.web = new WebDriver()
  }

  async leaveVault(): Promise<void> {
    this.web.forget()
    await this.scenario.removeNotes()
  }
}
