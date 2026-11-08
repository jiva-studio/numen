/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import path from 'node:path'
import type { Page } from 'playwright/test'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import { destroyNote } from '../../../e2e/desktop/tests/support/vault.js'

export class Scenario {
  private readonly paths = new Map<string, string>()
  private readonly createdPaths: string[] = []
  private hasWorkspace = false
  private vaultName: string | undefined

  constructor(readonly page: Page) {}

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    await openWorkspace(this.page)
    this.hasWorkspace = true
  }

  remember(title: string, notePath: string): void {
    this.paths.set(title, notePath)
    this.createdPaths.push(notePath)
  }

  getPath(how: string, value: string): string {
    if (how === 'at') return value
    if (how !== 'titled') throw new Error(`"${how}" does not name a tab or a note: use "at" or "titled"`)
    const notePath = this.paths.get(value)
    if (!notePath) throw new Error(`the scenario made no note titled "${value}"`)
    return notePath
  }

  getFolder(notePath: string): string {
    return path.posix.dirname(notePath).replace(/^\.$/, '')
  }

  bindVault(name: string): void {
    this.vaultName ??= name
    if (this.vaultName !== name) {
      throw new Error(`the run has one vault, named "${this.vaultName}" here, not "${name}"`)
    }
  }

  async removeNotes(): Promise<void> {
    for (const notePath of this.createdPaths) await destroyNote(this.page.request, notePath)
  }
}
