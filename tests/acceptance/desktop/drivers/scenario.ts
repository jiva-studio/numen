/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import type { Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { destroyNote } from '../../../e2e/desktop/tests/support/vault.js'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import { parseReference, pickByTitle } from './rules.js'

export class Scenario {
  private readonly paths = new Map<string, string>()
  private readonly titles = new Set<string>()
  private hasWorkspace = false
  private vaultName: string | undefined

  constructor(readonly page: Page) {}

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    await openWorkspace(this.page)
    this.hasWorkspace = true
  }

  async useTitle(title: string): Promise<void> {
    this.titles.add(title)
    await destroyNote(this.page.request, `notes/${title}.md`).catch(() => undefined)
  }

  remember(title: string, notePath: string): void {
    this.paths.set(title, notePath)
  }

  async getPath(how: string, value: string): Promise<string> {
    const reference = parseReference(how, value)
    if (reference.how === 'at') return value
    const known = this.paths.get(value)
    if (known) return known
    return pickByTitle(value, [...(await readSavedStates(this.page.request)).keys()])
  }

  bindVault(name: string): void {
    this.vaultName ??= name
    if (this.vaultName !== name) {
      throw new Error(`the run has one vault, named "${this.vaultName}" here, not "${name}"`)
    }
  }

  async removeNotes(): Promise<void> {
    const paths = [...this.paths.values(), ...[...this.titles].map((title) => `notes/${title}.md`)]
    for (const notePath of paths) await destroyNote(this.page.request, notePath).catch(() => undefined)
  }
}
