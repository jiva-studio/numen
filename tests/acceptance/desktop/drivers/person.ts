/**
 * The person at the window. Every method is one thing a person does or one outcome the product
 * holds; how it is done on the screen (the tabs, the keys, the labels) stays here, so a step and a
 * scenario say what happens and never how it looks.
 */
import { expect, type Locator, type Page, type Route } from 'playwright/test'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import {
  createNote,
  createUniqueTitle,
  destroyNote,
  getNotePanel,
  openNote,
  readNote,
  typeAtEnd,
  waitForSavedBody,
  writeNote,
} from '../../../e2e/desktop/tests/support/vault.js'

const WRITE_NOTE = '**/numen.v1.NoteService/WriteNote'
const CHANGED_ON_DISK = 'Saving paused: the file changed on disk.'
const REMOVED_FROM_VAULT = 'the note was removed from the vault'

/** The note the scenario works with. */
interface Subject {
  path: string
  initialText: string
  text: Locator
}

/** The saves of the window that wait until another program has changed the file. */
interface PausedSaves {
  routes: Route[]
  isPaused: boolean
}

export class Person {
  private subject?: Subject
  private pausedSaves?: PausedSaves
  private readonly createdPaths: string[] = []

  constructor(private readonly page: Page) {}

  private getSubject(): Subject {
    if (!this.subject) throw new Error('the scenario has no open note')
    return this.subject
  }

  private getBanner(): Locator {
    return getNotePanel(this.page, this.getSubject().path).getByRole('alert')
  }

  private async pauseSaves(): Promise<PausedSaves> {
    const paused: PausedSaves = { routes: [], isPaused: true }
    await this.page.route(WRITE_NOTE, async (route) => {
      if (paused.isPaused) paused.routes.push(route)
      else await route.continue()
    })
    return paused
  }

  private async resumeSaves(paused: PausedSaves): Promise<void> {
    const reply = this.page.waitForResponse((response) => response.url().includes('NoteService/WriteNote'))
    paused.isPaused = false
    for (const route of paused.routes) await route.continue()
    await reply
  }

  // Notes

  async openNote(title: string, initialText: string): Promise<void> {
    const path = await createNote(this.page.request, createUniqueTitle(title))
    this.createdPaths.push(path)
    await writeNote(this.page.request, path, `${initialText}\n`)
    await openWorkspace(this.page)
    const text = await openNote(this.page, path)
    await expect(text).toContainText(initialText)
    this.subject = { path, initialText, text }
  }

  async writeWithoutSaving(draft: string): Promise<void> {
    const subject = this.getSubject()
    this.pausedSaves = await this.pauseSaves()
    await typeAtEnd(this.page, subject.text, ` ${draft}`)
    await expect.poll(() => this.pausedSaves?.routes.length ?? 0).toBeGreaterThan(0)
  }

  async changeNoteOnDisk(text: string): Promise<void> {
    const subject = this.getSubject()
    await writeNote(this.page.request, subject.path, `${text}\n`)
    if (this.pausedSaves) {
      const paused = this.pausedSaves
      this.pausedSaves = undefined
      await this.resumeSaves(paused)
    }
  }

  async removeNoteFromDisk(): Promise<void> {
    await destroyNote(this.page.request, this.getSubject().path)
  }

  async keepOwnVersion(): Promise<void> {
    await this.getBanner().getByRole('button', { name: 'Keep my version' }).click()
  }

  async takeDiskVersion(): Promise<void> {
    await this.getBanner().getByRole('button', { name: 'Use disk version' }).click()
  }

  async recreateNote(): Promise<void> {
    await this.getBanner().getByRole('button', { name: 'Recreate it' }).click()
  }

  async leaveVault(): Promise<void> {
    if (this.pausedSaves) {
      const paused = this.pausedSaves
      this.pausedSaves = undefined
      paused.isPaused = false
      for (const route of paused.routes) await route.continue().catch(() => undefined)
    }
    await this.page.unroute(WRITE_NOTE).catch(() => undefined)
    for (const path of this.createdPaths) await destroyNote(this.page.request, path)
  }

  // Outcomes

  async expectNoteHas(text: string): Promise<void> {
    await expect(this.getSubject().text).toContainText(text)
  }

  async expectNoteLacks(text: string): Promise<void> {
    await expect(this.getSubject().text).not.toContainText(text)
  }

  async expectConflictReported(): Promise<void> {
    await expect(this.getBanner()).toContainText(CHANGED_ON_DISK)
  }

  async expectRemovalReported(): Promise<void> {
    await expect(this.getBanner()).toContainText(REMOVED_FROM_VAULT)
  }

  async expectFileHas(text: string): Promise<void> {
    await waitForSavedBody(this.page.request, this.getSubject().path, text)
  }

  async expectFileLacks(text: string): Promise<void> {
    expect((await readNote(this.page.request, this.getSubject().path)).body).not.toContain(text)
  }
}
