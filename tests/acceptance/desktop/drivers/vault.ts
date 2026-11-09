/** The vault of the scenario as a person finds it: what is in its folder before the application starts, and what is there after. */
import { expect } from 'playwright/test'
import { listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { assertEntry } from './rules.js'
import type { Scenario } from './scenario.js'

export class VaultDriver {
  constructor(private readonly scenario: Scenario) {}

  createEmpty(name: string): void {
    this.scenario.createVault(name)
  }

  writeNote(vault: string, title: string, text: string): void {
    this.scenario.getWorld().writeNote(this.scenario.getVault(vault), title, text)
  }

  makeFolder(vault: string, folder: string): void {
    this.scenario.getWorld().makeFolder(this.scenario.getVault(vault), folder)
  }

  copyPdf(vault: string, filePath: string): void {
    this.scenario.getWorld().copyPdf(this.scenario.getVault(vault), filePath)
  }

  async expectContains(vault: string, entry: string, entryPath: string): Promise<void> {
    if (this.scenario.hasWorld) {
      const world = this.scenario.getWorld()
      const name = this.scenario.getVault(vault)
      await expect.poll(() => world.getEntryKind(name, entryPath)).toBeDefined()
      assertEntry(entry, entryPath, world.getEntryKind(name, entryPath) === 'folder')
      return
    }
    this.scenario.setVault(vault)
    await expect.poll(() => this.listEntries(entryPath)).toContain(entryPath)
    const parent = entryPath.split('/').slice(0, -1).join('/')
    const found = (await listFiles(this.scenario.page.request, parent)).find((one) => one.path === entryPath)
    assertEntry(entry, entryPath, found?.isFolder ?? false)
  }

  async expectDoesNotContain(vault: string, file: string): Promise<void> {
    if (this.scenario.hasWorld) {
      const world = this.scenario.getWorld()
      const name = this.scenario.getVault(vault)
      await expect.poll(() => world.getEntryKind(name, file)).toBeUndefined()
      return
    }
    this.scenario.setVault(vault)
    await expect.poll(() => this.listEntries(file)).not.toContain(file)
  }

  private async listEntries(entryPath: string): Promise<string[]> {
    const parent = entryPath.split('/').slice(0, -1).join('/')
    return (await listFiles(this.scenario.page.request, parent)).map((one) => one.path)
  }
}
