/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import type { Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { destroyNote, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import { parseReference, pickByTitle } from './rules.js'

export class Scenario {
  private readonly paths = new Map<string, string>()
  private readonly titles = new Set<string>()
  private hasWorkspace = false
  private filesBefore: readonly string[] = []
  private vaultName: string | undefined

  constructor(readonly page: Page) {}

  get isOpen(): boolean {
    return this.hasWorkspace
  }

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    this.filesBefore = await listFilePaths(this.page.request)
    await openWorkspace(this.page)
    this.hasWorkspace = true
  }

  async claimTitle(title: string): Promise<void> {
    this.titles.add(title)
    await destroyNote(this.page.request, `notes/${title}.md`).catch(() => undefined)
  }

  rememberNote(title: string, notePath: string): void {
    this.paths.set(title, notePath)
  }

  async getPath(reference: string, name: string): Promise<string> {
    if (parseReference(reference, name).reference === 'at') return name
    const savedPath = this.paths.get(name)
    if (savedPath) return savedPath
    return pickByTitle(name, [...(await readSavedStates(this.page.request)).keys()])
  }

  setVault(name: string): void {
    this.vaultName ??= name
    if (this.vaultName !== name) {
      throw new Error(`the run has one vault, named "${this.vaultName}" here, not "${name}"`)
    }
  }

  async removeNotes(): Promise<void> {
    const made = this.hasWorkspace
      ? (await listFilePaths(this.page.request)).filter((one) => !this.filesBefore.includes(one))
      : []
    const paths = [...this.paths.values(), ...[...this.titles].map((title) => `notes/${title}.md`), ...made]
    for (const notePath of paths) await destroyNote(this.page.request, notePath).catch(() => undefined)
  }
}
