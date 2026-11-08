/** The person at the window as a whole: starting it, running a command, looking at the vault. */
import { expect } from 'playwright/test'
import { readActivePath } from '../../../e2e/desktop/tests/support/test.js'
import {
  getPaletteField,
  runPaletteCommand,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { createUniqueTitle, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import type { Scenario } from './scenario.js'

export class ApplicationDriver {
  constructor(private readonly scenario: Scenario) {}

  async startWindow(): Promise<void> {
    const { page } = this.scenario
    await this.scenario.ensureWorkspace()
    await page.reload()
    await waitForWindow(page)
  }

  async runCommand(command: string, kind: string, value: string): Promise<void> {
    if (kind !== 'name') throw new Error(`the value kind "${kind}" is not built yet`)
    if (command !== 'Create note') throw new Error(`the command "${command}" is not built yet`)
    await this.createNote(value)
  }

  async expectVaultHasNote(vaultName: string, how: string, value: string): Promise<void> {
    this.scenario.bindVault(vaultName)
    const notePath = this.scenario.getPath(how, value)
    const { request } = this.scenario.page
    await expect
      .poll(() => listFilePaths(request, this.scenario.getFolder(notePath)))
      .toContain(notePath)
  }

  private async createNote(name: string): Promise<void> {
    const { page } = this.scenario
    const title = createUniqueTitle(name)
    await this.scenario.ensureWorkspace()
    await runPaletteCommand(page, 'Create note', 'Create note')
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    await getPaletteField(page).fill(title)
    await expect(page.getByRole('option', { selected: true })).toContainText(title)
    await page.keyboard.press('Enter')
    await expect.poll(() => readActivePath(page.request)).toMatch(new RegExp(`${title}\\.md$`))
    this.scenario.remember(name, await readActivePath(page.request))
  }
}
