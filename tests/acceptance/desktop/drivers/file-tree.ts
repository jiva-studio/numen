/** The person at the file tree: finding a file or a folder in it, moving it, running a command on it. */
import { expect, type Locator } from 'playwright/test'
import { dragRowOnto } from '../../../e2e/desktop/tests/support/layout.js'
import { listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { FileTree } from './components/FileTree.js'
import { assertEntry, getAncestors } from './rules.js'
import type { Scenario } from './scenario.js'

export class FileTreeDriver {
  constructor(private readonly scenario: Scenario) {}

  private get tree(): FileTree {
    return new FileTree(this.scenario.page)
  }

  private async getRow(entryPath: string): Promise<Locator> {
    await this.scenario.ensureWorkspace()
    await this.scenario.closePalette()
    await this.tree.showTab()
    for (const folder of getAncestors(entryPath)) await this.tree.expandFolder(folder)
    const row = this.tree.getRow(entryPath)
    await expect(row).toBeVisible()
    return row
  }

  private async getEntryRow(entry: string, entryPath: string): Promise<Locator> {
    const row = await this.getRow(entryPath)
    const parent = entryPath.split('/').slice(0, -1).join('/')
    const found = (await listFiles(this.scenario.page.request, parent)).find((one) => one.path === entryPath)
    if (!found) throw new Error(`the vault has no file or folder at "${entryPath}"`)
    assertEntry(entry, entryPath, found.isFolder)
    return row
  }

  async open(entry: string, entryPath: string): Promise<void> {
    const row = await this.getEntryRow(entry, entryPath)
    // Enter folds a folder that is open already, as the folders above an opened file are.
    await this.tree.expand(row)
  }

  async move(filePath: string, folderPath: string): Promise<void> {
    const file = await this.getEntryRow('file', filePath)
    const folder = await this.getEntryRow('folder', folderPath)
    await dragRowOnto(this.scenario.page, file, folder)
  }

  async runCommandOn(label: string, entry: string, entryPath: string): Promise<void> {
    await this.tree.openMenu(await this.getEntryRow(entry, entryPath))
    const item = this.tree.getMenuItem(label)
    await expect(this.tree.getFirstMenuItem()).toBeVisible()
    if ((await item.count()) === 0) {
      const group = this.tree.getSubmenuItem()
      await expect(group).toHaveCount(1)
      await group.focus()
      await group.press('ArrowRight')
    }
    await item.first().click()
  }
}
