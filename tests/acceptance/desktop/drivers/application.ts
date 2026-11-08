/** The person at the application as a whole: starting it, running a command, looking at the vault. */
import fs from 'node:fs'
import path from 'node:path'
import { expect } from 'playwright/test'
import {
  getPaletteField as getPaletteInput,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { getVaultPath, listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { assertApplication, getEscaped, getPaletteField, parseCount, readNoteType } from './rules.js'
import type { Scenario } from './scenario.js'

export class ApplicationDriver {
  constructor(private readonly scenario: Scenario) {}

  async startApplication(name: string): Promise<void> {
    assertApplication(name)
    const { page } = this.scenario
    if (this.scenario.isOpen) {
      await page.reload()
    } else {
      await this.scenario.ensureWorkspace()
    }
    await waitForWindow(page)
  }

  async runCommand(label: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    const { page } = this.scenario
    const field = page.locator('input[data-palette="field"]').first()
    const name = new RegExp('^' + getEscaped(label) + '$')
    const command = page
      .getByRole('option')
      .filter({ has: page.locator('[data-palette="name"]', { hasText: name }) })
      .first()
    await page.keyboard.press('Control+p')
    await expect(field).toBeVisible()
    await field.fill(label)
    await command.click()
  }

  async enter(value: string): Promise<void> {
    const { page } = this.scenario
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    const field = getPaletteInput(page)
    const kind = getPaletteField((await field.getAttribute('placeholder')) ?? '')
    if (kind === 'item') {
      await page.getByRole('option', { name: new RegExp(`^${getEscaped(value)}$`) }).click()
    } else {
      if (kind === 'name') await this.scenario.claimTitle(value)
      await field.fill(value)
      if (kind === 'name') await expect(page.getByRole('option', { selected: true })).toContainText(value)
      await page.keyboard.press('Enter')
    }
    await expect(field).toBeHidden()
  }

  async expectVaultHasNote(vaultName: string, reference: string, name: string): Promise<void> {
    this.scenario.setVault(vaultName)
    if (reference === 'titled') {
      await expect.poll(() => this.findNote(name)).not.toBe('')
      this.scenario.rememberNote(name, await this.findNote(name))
      return
    }
    const notePath = await this.scenario.getPath(reference, name)
    const folder = notePath.split('/').slice(0, -1).join('/')
    await expect.poll(() => listFilePaths(this.scenario.page.request, folder)).toContain(notePath)
  }

  async expectNoteCount(type: string, vaultName: string, count: string): Promise<void> {
    this.scenario.setVault(vaultName)
    const expected = parseCount(count)
    await expect.poll(() => this.listNotesOfType(getVaultPath(), type).length).toBe(expected)
  }

  private listNotesOfType(folder: string, type: string): string[] {
    return fs.readdirSync(folder, { withFileTypes: true }).flatMap((entry) => {
      const full = path.join(folder, entry.name)
      if (entry.isDirectory()) return entry.name.startsWith('.') ? [] : this.listNotesOfType(full, type)
      if (!entry.name.endsWith('.md')) return []
      return readNoteType(fs.readFileSync(full, 'utf8')) === type ? [full] : []
    })
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
