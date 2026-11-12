/** The text of a note in its panel: typing at the end, replacing words, links and the banner. */
import { expect, type Locator } from 'playwright/test'
import { escapeRegExp } from '../rules.js'
import type { Host } from './host.js'
import { TabPanels } from './TabPanel.js'

export class NoteEditor {
  private readonly panels: TabPanels

  constructor(
    private readonly host: Host,
    private readonly title: string,
  ) {
    this.panels = new TabPanels(host)
  }

  panel(): Locator {
    return this.panels.ofNote(this.title)
  }

  text(): Locator {
    return this.panel().locator('.note__text .cm-content')
  }

  /** The alert the note shows above its text. */
  banner(): Locator {
    return this.panel().getByRole('alert')
  }

  /** The first place in the text that has the words of a link. */
  link(name: string): Locator {
    return this.text().getByText(name, { exact: false }).first()
  }

  /** The person clicks the text, goes to its end and types. */
  async typeAtEnd(typed: string): Promise<void> {
    await this.text().click()
    await this.host.page.keyboard.press('ControlOrMeta+End')
    await this.host.page.keyboard.type(typed)
  }

  /** The person selects the whole text of the note and types it again with the words replaced. */
  async replace(from: string, to: string): Promise<void> {
    const text = this.text()
    await text.click()
    const current = (await text.innerText()).replace(/\n$/, '')
    if (!current.includes(from)) throw new Error(`the note has no "${from}" to replace: it reads "${current}"`)
    await this.host.page.keyboard.press('ControlOrMeta+a')
    await this.host.page.keyboard.type(current.replace(from, to))
  }

  /** The first completion of the open list that begins with the label, once it shows. */
  async getCompletion(label: string): Promise<Locator> {
    const option = this.host.page
      .locator('.cm-tooltip-autocomplete')
      .getByRole('option', { name: new RegExp(`^${escapeRegExp(label)}`) })
      .first()
    await expect(option).toBeVisible()
    return option
  }
}
