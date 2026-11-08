/** The person at the application as a whole: starting it, running a command, looking at the vault. */
import fs from 'node:fs'
import path from 'node:path'
import { expect } from 'playwright/test'
import {
  getPaletteField,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { getVaultPath, listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { assertApplication, getEscaped, parseAsk, parseCount, readNoteType } from './rules.js'
import type { Scenario } from './scenario.js'

/** A command that depends on a download is listed within this time of the download. */
const COMMAND_LIMIT_MS = 30_000

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
    const command = page
      .getByRole('option')
      .filter({ has: page.locator('[data-palette="name"]', { hasText: new RegExp(`^${getEscaped(label)}/** The person at the application as a whole: starting it, running a command, looking at the vault. */
import fs from 'node:fs'
import path from 'node:path'
import { expect } from 'playwright/test'
import {
  getPaletteField,
  runPaletteCommand,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { getVaultPath, listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { assertApplication, getEscaped, parseAsk, parseCount, readNoteType } from './rules.js'
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

) }) })
      .first()
    // The palette lists the commands the open tab has when it opens: a command that a download adds is listed on the next opening.
    await expect(async () => {
      if (await field.isVisible()) await page.keyboard.press('Escape')
      await page.keyboard.press('Control+p')
      await expect(field).toBeVisible()
      await field.fill(label)
      await expect(command).toBeVisible({ timeout: 3_000 })
    }).toPass({ timeout: COMMAND_LIMIT_MS })
    await command.click()
  }

  async enter(value: string): Promise<void> {
    const { page } = this.scenario
    await expect(page.locator('[data-palette="crumb"]')).toBeVisible()
    const field = getPaletteField(page)
    const ask = parseAsk((await field.getAttribute('placeholder')) ?? '')
    if (ask === 'item') {
      await page.getByRole('option', { name: new RegExp(`^${getEscaped(value)}$`) }).click()
    } else {
      if (ask === 'name') await this.scenario.useTitle(value)
      await field.fill(value)
      if (ask === 'name') await expect(page.getByRole('option', { selected: true })).toContainText(value)
      await page.keyboard.press('Enter')
    }
    await expect(field).toBeHidden()
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

  async expectNoteCount(type: string, vaultName: string, count: string): Promise<void> {
    this.scenario.bindVault(vaultName)
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
