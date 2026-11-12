/** The person at the application as a whole: starting it, running a command, looking at the vault. */
import { expect, type Locator } from 'playwright/test'
import { waitForWindow } from '../../../e2e/desktop/tests/support/workspace.js'
import { Palette } from './components/Palette.js'
import type { JsonValue } from '../../../e2e/desktop/tests/support/json.js'
import { readActivePath } from '../../../e2e/desktop/tests/support/test.js'
import { callRPC, listFiles, listFilePaths } from '../../../e2e/desktop/tests/support/vault.js'
import {
  assertPaletteField,
  escapeRegExp,
  getPaletteField,
  parseApplication,
  type PaletteAction,
  type PaletteField,
} from './rules.js'
import type { Scenario } from './scenario.js'
import { hasMadeText } from './text-versions.js'

const COMMAND_PLACEHOLDER = 'Type a command'
const DELETE_TEXT = 'Delete text'
const DELETE_TEXT_ID = 'app:version.delete'
const PROOFREAD_TEXT = 'Proofread text'
/** A plugin may take this long to start before the first command of it answers. */
const PLUGIN_START_LIMIT_MS = 35_000
/** A command the palette offers as the second action of another row (Shift and Enter), by the row it belongs to. */
const SECOND_ACTIONS: Readonly<Record<string, string | undefined>> = { 'Open note beside': 'Open note' }

export class ApplicationDriver {
  private readonly palette: Palette

  constructor(private readonly scenario: Scenario) {
    this.palette = new Palette(scenario)
  }

  async startApplication(application: string): Promise<void> {
    const name = parseApplication(application)
    if (this.scenario.hasWorld) await this.scenario.startWorld(name)
    const { page } = this.scenario
    if (name === 'flashcards') {
      await page.goto(this.scenario.hasWorld ? '/' : `https://127.0.0.1:${process.env['NUMEN_FLASHCARDS_TEST_PORT']}/`)
      return
    }
    if (this.scenario.isOpen) {
      await page.reload()
    } else {
      await this.scenario.ensureWorkspace()
    }
    await waitForWindow(page)
  }

  /** Ends the application and starts it again over the same vault and settings. */
  async restartApplication(application: string): Promise<void> {
    const name = parseApplication(application)
    if (!this.scenario.hasServer) {
      throw new Error(`the application "${name}" has not started, so it cannot restart: begin with the application "${name}" starts`)
    }
    await this.startApplication(name)
  }

  async runCommand(label: string): Promise<void> {
    await this.scenario.ensureWorkspace()
    this.scenario.noteAction()
    const { page } = this.scenario
    const field = this.palette.field()
    const row = SECOND_ACTIONS[label]
    const name = new RegExp('^' + escapeRegExp(row ?? label) + '$')
    const command = this.palette.rowNamed(name)
    if (label === DELETE_TEXT) await this.expectOpenTabHasText()
    if (label === PROOFREAD_TEXT) await this.expectActiveFileHasMadeText()
    await page.keyboard.press('Control+p')
    await expect(field).toBeVisible()
    await field.fill(row ?? label)
    await expect(command).toHaveAttribute('data-here', 'true')
    if (row === undefined) await command.click()
    else await page.keyboard.press('Shift+Enter')
    const crumb = this.palette.crumb()
    await expect
      .poll(async () => !(await field.isVisible()) || ((await crumb.textContent()) ?? '').includes(label), {
        timeout: PLUGIN_START_LIMIT_MS,
      })
      .toBe(true)
  }

  async pickItem(value: string): Promise<void> {
    const field = this.palette.searchField()
    const kind = await this.readPaletteKind(field)
    assertPaletteField('pick', kind)
    const name = new RegExp(`^${escapeRegExp(value)}$`)
    if (kind === 'item') {
      await this.palette.optionNamed(name).click()
    } else {
      await field.fill(value)
      await this.palette.rowNamed(name).first().click()
    }
    await expect(field).toBeHidden()
  }

  async enterText(value: string): Promise<void> {
    const { page } = this.scenario
    const field = await this.readPaletteInput('type')
    await field.fill(value)
    await page.keyboard.press('Enter')
    await expect(field).toBeHidden()
  }

  private async readPaletteInput(action: PaletteAction): Promise<Locator> {
    const field = this.palette.searchField()
    assertPaletteField(action, await this.readPaletteKind(field))
    return field
  }

  private async readPaletteKind(field: Locator): Promise<PaletteField> {
    // The palette that took the command asks for its input once its placeholder is no longer the command prompt.
    await expect(field).not.toHaveAttribute('placeholder', COMMAND_PLACEHOLDER)
    return getPaletteField((await field.getAttribute('placeholder')) ?? '')
  }

  async expectVaultContainsNote(vaultName: string, reference: string, name: string): Promise<void> {
    if (this.scenario.hasWorld) this.scenario.getVault(vaultName)
    else this.scenario.checkSharedVault(vaultName)
    if (reference === 'titled') {
      await expect.poll(() => this.findNote(name)).not.toBe('')
      this.scenario.rememberNote(name, await this.findNote(name))
      return
    }
    const notePath = await this.scenario.getPath(reference, name)
    const folder = notePath.split('/').slice(0, -1).join('/')
    await expect.poll(() => listFilePaths(this.scenario.page.request, folder)).toContain(notePath)
  }

  private async expectActiveFileHasMadeText(): Promise<void> {
    const { request } = this.scenario.page
    await expect.poll(async () => hasMadeText(request, await readActivePath(request))).toBe(true)
  }

  private async expectOpenTabHasText(): Promise<void> {
    const { page } = this.scenario
    await expect
      .poll(async () => {
        const notePath = await readActivePath(page.request)
        const answer = await callRPC(page.request, 'CommandService/ListCommandItems', {
          path: notePath,
          commands: [{ command: DELETE_TEXT_ID }],
        })
        const lists = (answer['lists'] as { items?: JsonValue[] }[] | undefined) ?? []
        return lists.flatMap((one) => one.items ?? []).length
      })
      .toBeGreaterThan(0)
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
