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

/** The warnings a note gives, by the name a scenario uses. */
const WARNINGS: Record<string, string> = {
  change: 'Saving paused: the file changed on disk.',
  removal: 'the note was removed from the vault',
}

/** The ways out of a warning, by the name a scenario uses. */
const BANNER_CHOICES: Record<string, string> = {
  'keep my text': 'Keep my version',
  'use the saved text': 'Use disk version',
  'save the note again': 'Recreate it',
}

/** A note of the scenario, found by the title the scenario gives it. */
interface Subject {
  path: string
  text: Locator
}

/** The saves of a note that wait until another program has changed its file. */
interface PausedSaves {
  routes: Route[]
  isPaused: boolean
}

export class Person {
  private readonly subjects = new Map<string, Subject>()
  private readonly pausedSaves = new Map<string, PausedSaves>()
  private readonly createdPaths: string[] = []
  private hasWorkspace = false

  constructor(private readonly page: Page) {}

  private getSubject(title: string): Subject {
    const subject = this.subjects.get(title)
    if (!subject) throw new Error(`the scenario has no open note "${title}"`)
    return subject
  }

  private getBanner(title: string): Locator {
    return getNotePanel(this.page, this.getSubject(title).path).getByRole('alert')
  }

  private async pauseSaves(path: string): Promise<PausedSaves> {
    const paused: PausedSaves = { routes: [], isPaused: true }
    await this.page.route(WRITE_NOTE, async (route) => {
      if (!route.request().postData()?.includes(path)) await route.fallback()
      else if (paused.isPaused) paused.routes.push(route)
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

  // Things a person does and things that happen to the notes

  async openNote(title: string, initialText: string): Promise<void> {
    const path = await createNote(this.page.request, createUniqueTitle(title))
    this.createdPaths.push(path)
    await writeNote(this.page.request, path, `${initialText}\n`)
    if (!this.hasWorkspace) await openWorkspace(this.page)
    this.hasWorkspace = true
    const text = await openNote(this.page, path)
    await expect(text).toContainText(initialText)
    this.subjects.set(title, { path, text })
  }

  async writeWithoutSaving(draft: string, title: string): Promise<void> {
    const subject = this.getSubject(title)
    const paused = await this.pauseSaves(subject.path)
    this.pausedSaves.set(title, paused)
    await typeAtEnd(this.page, subject.text, ` ${draft}`)
    await expect.poll(() => paused.routes.length).toBeGreaterThan(0)
  }

  async changeNoteOnDisk(text: string, title: string): Promise<void> {
    await writeNote(this.page.request, this.getSubject(title).path, `${text}\n`)
    const paused = this.pausedSaves.get(title)
    if (paused) {
      this.pausedSaves.delete(title)
      await this.resumeSaves(paused)
    }
  }

  async removeNoteFromDisk(title: string): Promise<void> {
    await destroyNote(this.page.request, this.getSubject(title).path)
  }

  async choose(option: string, title: string): Promise<void> {
    const name = BANNER_CHOICES[option]
    if (!name) throw new Error(`no choice "${option}"`)
    await this.getBanner(title).getByRole('button', { name }).click()
  }

  async leaveVault(): Promise<void> {
    for (const paused of this.pausedSaves.values()) {
      paused.isPaused = false
      for (const route of paused.routes) await route.continue().catch(() => undefined)
    }
    this.pausedSaves.clear()
    await this.page.unroute(WRITE_NOTE).catch(() => undefined)
    for (const path of this.createdPaths) await destroyNote(this.page.request, path)
  }

  // What the notes and the files hold

  async expectHas(kind: string, title: string, text: string): Promise<void> {
    const subject = this.getSubject(title)
    if (kind === 'saved') await waitForSavedBody(this.page.request, subject.path, text)
    else await expect(subject.text).toContainText(text)
  }

  async expectLacks(kind: string, title: string, text: string): Promise<void> {
    const subject = this.getSubject(title)
    if (kind === 'saved') {
      expect((await readNote(this.page.request, subject.path)).body).not.toContain(text)
    } else {
      await expect(subject.text).not.toContainText(text)
    }
  }

  async expectWarning(title: string, kind: string): Promise<void> {
    const message = WARNINGS[kind]
    if (!message) throw new Error(`no warning "${kind}"`)
    await expect(this.getBanner(title)).toContainText(message)
  }
}
