/** The person at the file tree: finding a file or a folder in it, moving it, running a command on it. */
import { expect, type Locator } from 'playwright/test'
import { dragRowOnto } from '../../../e2e/desktop/tests/support/layout.js'
import { listFiles } from '../../../e2e/desktop/tests/support/vault.js'
import { assertEntry, getAncestors } from './rules.js'
import type { Scenario } from './scenario.js'

export class FileTreeDriver {
  constructor(private readonly scenario: Scenario) {}

  private async getRow(entryPath: string): Promise<Locator> {
    const { page } = this.scenario
    await this.scenario.ensureWorkspace()
    await page.getByRole('tab', { name: /^Files/ }).click()
    for (const folder of getAncestors(entryPath)) {
      const parent = page.locator(`[data-tree-row="${folder}"]`)
      await expect(parent).toBeVisible()
      if ((await parent.getAttribute('aria-expanded')) !== 'true') await parent.press('Enter')
      await expect(parent).toHaveAttribute('aria-expanded', 'true')
    }
    const row = page.locator(`[data-tree-row="${entryPath}"]`)
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
    await row.press('Enter')
    await expect(row).toHaveAttribute('aria-expanded', 'true')
  }

  async move(filePath: string, folderPath: string): Promise<void> {
    const file = await this.getEntryRow('file', filePath)
    const folder = await this.getEntryRow('folder', folderPath)
    await dragRowOnto(this.scenario.page, file, folder)
  }

  async runCommandOn(label: string, entry: string, entryPath: string): Promise<void> {
    const { page } = this.scenario
    await (await this.getEntryRow(entry, entryPath)).click({ button: 'right' })
    const item = page.getByRole('menuitem', { name: label, exact: true })
    await expect(page.getByRole('menuitem').first()).toBeVisible()
    if ((await item.count()) === 0) {
      const group = page.getByRole('menuitem').and(page.locator('[aria-haspopup="menu"]'))
      await expect(group).toHaveCount(1)
      await group.focus()
      await group.press('ArrowRight')
    }
    await item.first().click()
  }
}
