/** The person at a note: writing it, choosing in its banner, and what the note holds in the window and on disk. */
import { expect, type Locator, type Route } from 'playwright/test'
import { getEscaped } from './tab-position.js'
import {
  createNote,
  createUniqueTitle,
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

const WRITE_NOTE = '**/numen.v1.NoteService/WriteNote'

interface PausedSaves {
  routes: Route[]
  isPaused: boolean
}

export class NoteDriver {
  private readonly pausedSaves = new Map<string, PausedSaves>()

  constructor(private readonly scenario: Scenario) {}

  private getText(notePath: string): Locator {
    return getNoteText(this.scenario.page, notePath)
  }

  private getPanel(notePath: string): Locator {
    return getNotePanel(this.scenario.page, notePath)
  }

  private async pauseSaves(notePath: string): Promise<PausedSaves> {
    const paused: PausedSaves = { routes: [], isPaused: true }
    await this.scenario.page.route(WRITE_NOTE, async (route) => {
      if (!route.request().postData()?.includes(notePath)) await route.fallback()
      else if (paused.isPaused) paused.routes.push(route)
      else await route.continue()
    })
    return paused
  }

  private async resumeSaves(paused: PausedSaves): Promise<void> {
    const reply = this.scenario.page.waitForResponse((response) =>
      response.url().includes('NoteService/WriteNote'),
    )
    paused.isPaused = false
    for (const route of paused.routes) await route.continue()
    await reply
  }

  async create(title: string, initialText: string): Promise<void> {
    const { page } = this.scenario
    const notePath = await createNote(page.request, createUniqueTitle(title))
    this.scenario.remember(title, notePath)
    await writeNote(page.request, notePath, `${initialText}\n`)
    await this.scenario.ensureWorkspace()
    await expect(await openNote(page, notePath)).toContainText(initialText)
  }

  async write(how: string, value: string, text: string): Promise<void> {
    const notePath = this.scenario.getPath(how, value)
    await typeAtEnd(this.scenario.page, this.getText(notePath), ` ${text}`)
  }

  async writeWithoutSaving(how: string, value: string, text: string): Promise<void> {
    const notePath = this.scenario.getPath(how, value)
    const paused = await this.pauseSaves(notePath)
    this.pausedSaves.set(notePath, paused)
    await this.write(how, value, text)
    await expect.poll(() => paused.routes.length).toBeGreaterThan(0)
  }

  async changeOnDisk(how: string, value: string, text: string): Promise<void> {
    const notePath = this.scenario.getPath(how, value)
    await writeNote(this.scenario.page.request, notePath, `${text}\n`)
    const paused = this.pausedSaves.get(notePath)
    if (!paused) return
    this.pausedSaves.delete(notePath)
    await this.resumeSaves(paused)
  }

  async removeFromDisk(how: string, value: string): Promise<void> {
    await destroyNote(this.scenario.page.request, this.scenario.getPath(how, value))
  }

  async choose(label: string, how: string, value: string): Promise<void> {
    const banner = this.getPanel(this.scenario.getPath(how, value)).getByRole('alert')
    await banner.getByRole('button', { name: label }).click()
  }

  async chooseCompletion(label: string, how: string, value: string): Promise<void> {
    this.scenario.getPath(how, value)
    const option = this.scenario.page
      .locator('.cm-tooltip-autocomplete')
      .getByRole('option', { name: new RegExp(`^${getEscaped(label)}`) })
      .first()
    await expect(option).toBeVisible()
    await option.click()
  }

  async openLink(link: string, how: string, value: string): Promise<void> {
    const name = link.replace(/^\[\[|\]\]$/g, '')
    await this.getText(this.scenario.getPath(how, value))
      .getByText(name, { exact: false })
      .first()
      .click({ modifiers: ['ControlOrMeta'] })
  }

  async expectHas(how: string, value: string, text: string): Promise<void> {
    await expect(this.getPanel(this.scenario.getPath(how, value))).toContainText(text)
  }

  async expectLacks(how: string, value: string, text: string): Promise<void> {
    await expect(this.getPanel(this.scenario.getPath(how, value))).not.toContainText(text)
  }

  async expectOnDiskHas(how: string, value: string, text: string): Promise<void> {
    await waitForSavedBody(this.scenario.page.request, this.scenario.getPath(how, value), text)
  }

  async expectLinks(how: string, value: string, toHow: string, toValue: string): Promise<void> {
    const from = this.scenario.getPath(how, value)
    const backlinks = (): Promise<string[]> =>
      listBacklinks(this.scenario.page.request, this.scenario.getPath(toHow, toValue))
    await expect.poll(backlinks).toContain(from)
  }

  async releaseSaves(): Promise<void> {
    for (const paused of this.pausedSaves.values()) {
      paused.isPaused = false
      for (const route of paused.routes) await route.continue().catch(() => undefined)
    }
    this.pausedSaves.clear()
    await this.scenario.page.unroute(WRITE_NOTE).catch(() => undefined)
  }
}
