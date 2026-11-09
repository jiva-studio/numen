/** What one scenario shares between the levels of the driver: the page, the notes it made, its vault. */
import type { Page } from 'playwright/test'
import { readSavedStates } from '../../../e2e/desktop/tests/support/test.js'
import { callRPC, destroyNote, destroyTrashed, listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { openWorkspace } from '../../../e2e/desktop/tests/support/workspace.js'
import { parseReference, pickByTitle } from './rules.js'

const DELETE_TEXT_ID = 'app:version.delete'

export class Scenario {
  private readonly paths = new Map<string, string>()
  private readonly titles = new Set<string>()
  private readonly files = new Set<string>()
  private hasWorkspace = false
  private filesBefore: readonly string[] = []
  private vaultName: string | undefined

  constructor(readonly page: Page) {}

  get isOpen(): boolean {
    return this.hasWorkspace
  }

  async ensureWorkspace(): Promise<void> {
    if (this.hasWorkspace) return
    this.filesBefore = await this.listEntries()
    await openWorkspace(this.page)
    this.hasWorkspace = true
  }

  /** The paths of the files and folders of the vault below `folder`; the contents of a folder come before the folder. */
  async listEntries(folder = ''): Promise<string[]> {
    const entries = await listFiles(this.page.request, folder)
    const nested = await Promise.all(entries.filter((one) => one.isFolder).map((one) => this.listEntries(one.path)))
    return [...nested.flat(), ...entries.map((one) => one.path)]
  }

  async claimTitle(title: string): Promise<void> {
    this.titles.add(title)
    await destroyNote(this.page.request, `notes/${title}.md`).catch(() => undefined)
  }

  rememberFile(filePath: string): void {
    this.files.add(filePath)
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

  /** Deletes the text versions made over the files the scenario opened: a file of the vault outlives the scenario. */
  async removeTexts(): Promise<void> {
    for (const filePath of this.files) {
      const answer = await callRPC(this.page.request, 'CommandService/ListCommandItems', {
        path: filePath,
        commands: [{ command: DELETE_TEXT_ID }],
      }).catch(() => ({}))
      const [list] = (answer['lists'] as { items?: { id: string }[] }[] | undefined) ?? []
      for (const item of list?.items ?? []) {
        const result = await callRPC(this.page.request, 'CommandService/RunPluginCommand', {
          command: DELETE_TEXT_ID,
          path: filePath,
          itemId: item.id,
        })
        console.log('DBGDEL', JSON.stringify(item), JSON.stringify(result))
      }
    }
  }

  async removeNotes(): Promise<void> {
    await this.removeTexts()
    const made = this.hasWorkspace
      ? (await this.listEntries()).filter((one) => !this.filesBefore.includes(one))
      : []
    const paths = [...this.paths.values(), ...[...this.titles].map((title) => `notes/${title}.md`), ...made]
    for (const notePath of paths) await destroyNote(this.page.request, notePath).catch(() => undefined)
    for (const title of this.titles) destroyTrashed(`${title}.md`)
  }
}
