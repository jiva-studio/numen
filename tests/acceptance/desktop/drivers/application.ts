/** The person at the application as a whole: starting it, running a command, looking at the vault. */
import fs from 'node:fs'
import path from 'node:path'
import { expect } from 'playwright/test'
import {
  getPaletteField as getPaletteInput,
  waitForWindow,
} from '../../../e2e/desktop/tests/support/workspace.js'
import { callRPC, getVaultPath, listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import { assertApplication, assertEntry, getEscaped, getPaletteField, parseCount, readNoteType } from './rules.js'
import type { Scenario } from './scenario.js'

/** A text made by a plugin may take this long before the commands that work on it are listed. */
const TEXT_LIMIT_MS = 90_000
const DELETE_TEXT = 'Delete text'
const DELETE_TEXT_ID = 'app:version.delete'

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
    if (label === DELETE_TEXT) await this.expectOpenTabHasText()
    await page.keyboard.press('Control+p')
    await expect(field).toBeVisible()
    // A command that works on a text is listed once the file has one, so the label is typed again until it shows.
    await expect(async () => {
      await field.fill('')
      await field.fill(label)
      await expect(command).toHaveAttribute('data-here', 'true', { timeout: 3_000 })
    }).toPass({ timeout: TEXT_LIMIT_MS })
    await page.keyboard.press('Enter')
    const crumb = page.locator('[data-palette="crumb"]')
    await expect.poll(async () => !(await field.isVisible()) || ((await crumb.textContent()) ?? '').includes(label)).toBe(true)
  }

  async enter(value: string): Promise<void> {
    const { page } = this.scenario
    const treeField = page.getByRole('tree').getByRole('textbox')
    await expect(treeField.or(page.locator('[data-palette="crumb"]'))).toBeVisible()
    if (await treeField.isVisible()) {
      await treeField.fill(value)
      await treeField.press('Enter')
      await expect(treeField).toBeHidden()
      return
    }
    const field = getPaletteInput(page)
    const kind = getPaletteField((await field.getAttribute('placeholder')) ?? '')
    if (kind === 'item') {
      await page.getByRole('option', { name: new RegExp(`^${getEscaped(value)}$`) }).click()
    } else if (kind === 'note') {
      await field.fill(value)
      const name = new RegExp(`^${getEscaped(value)}$`)
      await page
        .getByRole('option')
        .filter({ has: page.locator('[data-palette="name"]', { hasText: name }) })
        .first()
        .click()
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

  async expectVaultHasEntry(vaultName: string, entry: string, entryPath: string): Promise<void> {
    this.scenario.setVault(vaultName)
    await expect.poll(() => this.listEntries(entryPath)).toContain(entryPath)
    const parent = entryPath.split('/').slice(0, -1).join('/')
    const found = (await listFiles(this.scenario.page.request, parent)).find((one) => one.path === entryPath)
    assertEntry(entry, entryPath, found?.isFolder ?? false)
  }

  async expectVaultDoesNotHaveFile(vaultName: string, file: string): Promise<void> {
    this.scenario.setVault(vaultName)
    await expect.poll(() => this.listEntries(file)).not.toContain(file)
  }

  private async listEntries(entryPath: string): Promise<string[]> {
    const parent = entryPath.split('/').slice(0, -1).join('/')
    return (await listFiles(this.scenario.page.request, parent)).map((one) => one.path)
  }

  async expectNoteCount(type: string, vaultName: string, count: string): Promise<void> {
    this.scenario.setVault(vaultName)
    const expected = parseCount(count)
    await expect.poll(() => this.listNotesOfType(getVaultPath(), type).length).toBe(expected)
  }

  private async expectOpenTabHasText(): Promise<void> {
    const { page } = this.scenario
    const closeButton = page.getByRole('tab', { selected: true }).getByRole('button').first()
    const title = ((await closeButton.getAttribute('aria-label')) ?? '').replace(/^Close /, '')
    const notePath = await this.findNote(title)
    await expect
      .poll(async () => {
        const answer = await callRPC(page.request, 'CommandService/ListCommandItems', {
          path: notePath,
          commands: [{ command: DELETE_TEXT_ID }],
        })
        const [list] = (answer['lists'] as { items?: unknown[] }[] | undefined) ?? []
        return list?.items?.length ?? 0
      })
      .toBeGreaterThan(0)
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
