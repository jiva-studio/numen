/** The person at the window as a whole: starting it, running a command, looking at the vault. */
import { expect } from 'playwright/test'
import {
  getPaletteField,
  runPaletteCommand,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { assertPaletteStep } from './rules.js'
import type { Scenario } from './scenario.js'

export class ApplicationDriver {
  constructor(private readonly scenario: Scenario) {}

  async startWindow(): Promise<void> {
    const { page } = this.scenario
    await this.scenario.ensureWorkspace()
    await page.reload()
    await waitForWindow(page)
  }

  async runCommand(label: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    await runPaletteCommand(this.scenario.page, label, label)
  }

  async enter(kind: string, value: string): Promise<void> {
    assertPaletteStep(kind)
    const { page } = this.scenario
    await this.scenario.useTitle(value)
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    await getPaletteField(page).fill(value)
    await expect(page.getByRole('option', { selected: true })).toContainText(value)
    await page.keyboard.press('Enter')
  }

  async expectVaultHasNote(vaultName: string, how: string, value: string): Promise<void> {
    this.scenario.bindVault(vaultName)
    if (how === 'titled') {
      await expect.poll(() => this.findNote(value)).not.toBe('')
      this.scenario.remember(value, await this.findNote(value))
      return
    }
    const notePath = await this.scenario.getPath(how, value)
    const folder = notePath.split('/').slice(0, -1).join('/')
    await expect.poll(() => listFilePaths(this.scenario.page.request, folder)).toContain(notePath)
  }

  private async findNote(title: string): Promise<string> {
    const { request } = this.scenario.page
    const folders = (await listFiles(request)).filter((one) => one.isFolder).map((one) => one.path)
    const ordered = ['', ...folders.sort((a, b) => Number(b === 'notes') - Number(a === 'notes'))]
    for (const folder of ordered) {
      const found = (await listFilePaths(request, folder)).find((one) => one.endsWith(`/${title}.md`) || one === `${title}.md`)
      if (found) return found
    }
    return ''
  }
}
