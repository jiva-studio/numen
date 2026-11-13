/** The person at a note: writing it, choosing in its banner, and what the note holds in the window and on disk. */
import { expect } from 'playwright/test'
import {
  getTabName,
  listBacklinks,
  openNote,
  waitForSavedBody,
  writeNote,
  destroyNote,
} from '../../../e2e/desktop/tests/support/vault.js'
import { getConnectionSides } from '../../../e2e/desktop/tests/support/layout.js'
import { NoteEditor } from './components/NoteEditor.js'
import { TabStrip } from './components/TabStrip.js'
import type { Scenario } from './scenario.js'

export class NoteDriver {
  private readonly strip: TabStrip

  constructor(private readonly scenario: Scenario) {
    this.strip = new TabStrip(scenario)
  }

  private getEditor(notePath: string): NoteEditor {
    return new NoteEditor(this.scenario, getTabName(notePath))
  }

  async open(title: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await this.scenario.closePalette()
    await openNote(this.scenario.page, this.scenario.getNotePath(title))
  }

  async write(actor: string, text: string, reference: string, name: string): Promise<void> {
    const notePath = await this.scenario.getPath(reference, name)
    if (actor === 'the person') {
      await this.scenario.closePalette()
      await this.getEditor(notePath).typeAtEnd(` ${text}`)
    } else if (actor === 'another program') {
      await writeNote(this.scenario.page.request, notePath, `${text}\n`)
    } else {
      throw new Error(`"${actor}" does not write notes: use "the person" or "another program"`)
    }
  }

  /** The person selects the whole text of the note and types it again with the words replaced. */
  async replace(from: string, to: string, reference: string, name: string): Promise<void> {
    const editor = this.getEditor(await this.scenario.getPath(reference, name))
    await this.scenario.closePalette()
    await editor.replace(from, to)
  }

  async removeFromDisk(reference: string, name: string): Promise<void> {
    await destroyNote(this.scenario.page.request, await this.scenario.getPath(reference, name))
  }

  async choose(label: string, reference: string, name: string): Promise<void> {
    const banner = this.getEditor(await this.scenario.getPath(reference, name)).banner()
    await banner.getByRole('button', { name: label }).click()
  }

  async chooseCompletion(label: string, reference: string, name: string): Promise<void> {
    const editor = this.getEditor(await this.scenario.getPath(reference, name))
    await (await editor.getCompletion(label)).click()
  }

  async openLink(link: string, reference: string, name: string): Promise<void> {
    const linkName = link.replace(/^\[\[|\]\]$/g, '')
    await this.getEditor(await this.scenario.getPath(reference, name))
      .link(linkName)
      .click({ modifiers: ['ControlOrMeta'] })
  }

  private async showNote(notePath: string): Promise<void> {
    await this.strip.show(this.strip.noteTab(getTabName(notePath)))
  }

  async expectHasText(reference: string, name: string, text: string): Promise<void> {
    const notePath = await this.scenario.getPath(reference, name)
    await this.scenario.closePalette()
    await this.showNote(notePath)
    await expect(this.getEditor(notePath).panel()).toContainText(text)
  }

  /** The cursor is on the line of the heading, which is where the note opens from a heading of the plex. */
  async expectAtHeading(reference: string, name: string, heading: string): Promise<void> {
    const notePath = await this.scenario.getPath(reference, name)
    await expect(this.getEditor(notePath).text().locator('.cm-activeLine')).toContainText(heading)
  }

  async expectDoesNotHaveText(reference: string, name: string, text: string): Promise<void> {
    await expect(this.getEditor(await this.scenario.getPath(reference, name)).panel()).not.toContainText(text)
  }

  async expectOnDiskHasText(reference: string, name: string, text: string): Promise<void> {
    await waitForSavedBody(this.scenario.page.request, await this.scenario.getPath(reference, name), text)
  }

  async expectLinks(reference: string, name: string, toReference: string, toName: string): Promise<void> {
    await expect.poll(() => this.isLinked(reference, name, toReference, toName)).toBe(true)
  }

  async expectDoesNotLink(reference: string, name: string, toReference: string, toName: string): Promise<void> {
    await expect.poll(() => this.isLinked(reference, name, toReference, toName)).toBe(false)
  }

  private async isLinked(reference: string, name: string, toReference: string, toName: string): Promise<boolean> {
    const from = await this.scenario.getPath(reference, name)
    const to = await this.scenario.getPath(toReference, toName)
    const { request } = this.scenario.page
    if ((await listBacklinks(request, to)).includes(from)) return true
    const sides = await getConnectionSides(request, from, to)
    return sides.first || sides.second
  }

}
