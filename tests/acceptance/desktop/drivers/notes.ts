/** The person at a note: writing it, choosing in its banner, and what the note holds in the window and on disk. */
import { expect, type Locator } from 'playwright/test'
import { getEscaped } from './rules.js'
import {
  getNotePanel,
  getNoteText,
  listBacklinks,
  openNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
  destroyNote,
} from '../../../e2e/desktop/tests/support/vault.js'
import { getConnectionSides } from '../../../e2e/desktop/tests/support/layout.js'
import type { Scenario } from './scenario.js'

export class NoteDriver {
  constructor(private readonly scenario: Scenario) {}

  private getText(notePath: string): Locator {
    return getNoteText(this.scenario.page, notePath)
  }

  private getPanel(notePath: string): Locator {
    return getNotePanel(this.scenario.page, notePath)
  }

  async open(title: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await openNote(this.scenario.page, this.scenario.getNotePath(title))
  }

  async write(actor: string, text: string, reference: string, name: string): Promise<void> {
    const notePath = await this.scenario.getPath(reference, name)
    if (actor === 'the person') {
      await typeAtEnd(this.scenario.page, this.getText(notePath), ` ${text}`)
    } else if (actor === 'another program') {
      await writeNote(this.scenario.page.request, notePath, `${text}\n`)
    } else {
      throw new Error(`"${actor}" does not write notes: use "the person" or "another program"`)
    }
  }

  async removeFromDisk(reference: string, name: string): Promise<void> {
    await destroyNote(this.scenario.page.request, await this.scenario.getPath(reference, name))
  }

  async choose(label: string, reference: string, name: string): Promise<void> {
    const banner = this.getPanel(await this.scenario.getPath(reference, name)).getByRole('alert')
    await banner.getByRole('button', { name: label }).click()
  }

  async chooseCompletion(label: string, reference: string, name: string): Promise<void> {
    await this.scenario.getPath(reference, name)
    const option = this.scenario.page
      .locator('.cm-tooltip-autocomplete')
      .getByRole('option', { name: new RegExp(`^${getEscaped(label)}`) })
      .first()
    await expect(option).toBeVisible()
    await option.click()
  }

  async openLink(link: string, reference: string, name: string): Promise<void> {
    const linkName = link.replace(/^\[\[|\]\]$/g, '')
    await this.getText(await this.scenario.getPath(reference, name))
      .getByText(linkName, { exact: false })
      .first()
      .click({ modifiers: ['ControlOrMeta'] })
  }

  async expectHasText(reference: string, name: string, text: string): Promise<void> {
    await expect(this.getPanel(await this.scenario.getPath(reference, name))).toContainText(text)
  }

  async expectDoesNotHaveText(reference: string, name: string, text: string): Promise<void> {
    await expect(this.getPanel(await this.scenario.getPath(reference, name))).not.toContainText(text)
  }

  async expectOnDiskHasText(reference: string, name: string, text: string): Promise<void> {
    await waitForSavedBody(this.scenario.page.request, await this.scenario.getPath(reference, name), text)
  }

  async expectLinks(reference: string, name: string, toReference: string, toName: string): Promise<void> {
    const from = await this.scenario.getPath(reference, name)
    const to = await this.scenario.getPath(toReference, toName)
    const { request } = this.scenario.page
    const isLinked = async (): Promise<boolean> => {
      if ((await listBacklinks(request, to)).includes(from)) return true
      const sides = await getConnectionSides(request, from, to)
      return sides.first || sides.second
    }
    await expect.poll(isLinked).toBe(true)
  }

}
