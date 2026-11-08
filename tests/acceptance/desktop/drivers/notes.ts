/** The person at a note: writing it, choosing in its banner, and what the note holds in the window and on disk. */
import { expect, type Locator } from 'playwright/test'
import { getEscaped } from './rules.js'
import {
  createNote,
  getNotePanel,
  getNoteText,
  listBacklinks,
  openNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
  destroyNote,
} from '../../../e2e/desktop/tests/support/vault.js'
import type { Scenario } from './scenario.js'

export class NoteDriver {
  constructor(private readonly scenario: Scenario) {}

  private getText(notePath: string): Locator {
    return getNoteText(this.scenario.page, notePath)
  }

  private getPanel(notePath: string): Locator {
    return getNotePanel(this.scenario.page, notePath)
  }

  async create(title: string, initialText: string): Promise<void> {
    const { page } = this.scenario
    await this.scenario.useTitle(title)
    const notePath = await createNote(page.request, title)
    this.scenario.remember(title, notePath)
    await writeNote(page.request, notePath, `${initialText}\n`)
    await this.scenario.ensureWorkspace()
    // The editor hides the brackets of a link where the cursor is not.
    await expect(await openNote(page, notePath)).toContainText(initialText.replace(/\[\[|\]\]/g, ''))
  }

  async write(actor: string, text: string, how: string, value: string): Promise<void> {
    const notePath = await this.scenario.getPath(how, value)
    if (actor === 'the person') {
      await typeAtEnd(this.scenario.page, this.getText(notePath), ` ${text}`)
    } else if (actor === 'another program') {
      await writeNote(this.scenario.page.request, notePath, `${text}\n`)
    } else {
      throw new Error(`"${actor}" does not write notes: use "the person" or "another program"`)
    }
  }

  async removeFromDisk(how: string, value: string): Promise<void> {
    await destroyNote(this.scenario.page.request, await this.scenario.getPath(how, value))
  }

  async choose(label: string, how: string, value: string): Promise<void> {
    const banner = this.getPanel(await this.scenario.getPath(how, value)).getByRole('alert')
    await banner.getByRole('button', { name: label }).click()
  }

  async chooseCompletion(label: string, how: string, value: string): Promise<void> {
    await this.scenario.getPath(how, value)
    const option = this.scenario.page
      .locator('.cm-tooltip-autocomplete')
      .getByRole('option', { name: new RegExp(`^${getEscaped(label)}`) })
      .first()
    await expect(option).toBeVisible()
    await option.click()
  }

  async openLink(link: string, how: string, value: string): Promise<void> {
    const name = link.replace(/^\[\[|\]\]$/g, '')
    await this.getText(await this.scenario.getPath(how, value))
      .getByText(name, { exact: false })
      .first()
      .click({ modifiers: ['ControlOrMeta'] })
  }

  async expectHas(how: string, value: string, text: string): Promise<void> {
    await expect(this.getPanel(await this.scenario.getPath(how, value))).toContainText(text)
  }

  async expectLacks(how: string, value: string, text: string): Promise<void> {
    await expect(this.getPanel(await this.scenario.getPath(how, value))).not.toContainText(text)
  }

  async expectOnDiskHas(how: string, value: string, text: string): Promise<void> {
    await waitForSavedBody(this.scenario.page.request, await this.scenario.getPath(how, value), text)
  }

  async expectLinks(how: string, value: string, toHow: string, toValue: string): Promise<void> {
    const from = await this.scenario.getPath(how, value)
    const to = await this.scenario.getPath(toHow, toValue)
    const backlinks = (): Promise<string[]> => listBacklinks(this.scenario.page.request, to)
    await expect.poll(backlinks).toContain(from)
  }

}
