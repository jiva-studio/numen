/**
 * The person at the arguments of a command: the palette asks for them one at a time, and a table of
 * name and value rows answers them in the order the palette asks. A row names what the palette asks
 * for with the words the person reads: the placeholder of the field, or the title of the list.
 */
import { expect } from 'playwright/test'
import { Palette } from './components/Palette.js'
import { escapeRegExp, getPaletteField, isAskedFor, parsePaletteRows } from './rules.js'
import type { Scenario } from './scenario.js'

const COMMAND_PLACEHOLDER = 'Type a command'
const ASK_LIMIT_MS = 15_000
/** The placeholder of the field of a palette that asks to choose one of the rows of a question. */
const CONFIRM_PLACEHOLDER = 'Choose'

interface Ask {
  readonly placeholder: string
  readonly groups: readonly string[]
  readonly isInTree: boolean
}

export class PaletteDriver {
  private readonly palette: Palette

  constructor(private readonly scenario: Scenario) {
    this.palette = new Palette(scenario)
  }

  /** Answers what the command asks for with the rows, and fails when the palette asks for something else. */
  async answer(rows: readonly (readonly [string, string])[]): Promise<void> {
    for (const [key, value] of rows) {
      const ask = await this.waitForAsk(key)
      await this.give(ask, key, value)
    }
    await this.expectNothingAsked()
  }

  /** The palette lists each row of the table, in the list it names, and shows it disabled or not as the table says. */
  async expectRows(rows: readonly (readonly string[])[]): Promise<void> {
    for (const { list, row, isDisabled } of parsePaletteRows(rows)) {
      await this.waitForAsk(list)
      const option = this.palette.rowNamed(new RegExp('^' + escapeRegExp(row) + '$'))
      await expect(option.first(), `the list "${list}" has no row "${row}"`).toBeVisible()
      if (isDisabled) await expect(option.first()).toHaveAttribute('aria-disabled', 'true')
      else await expect(option.first()).not.toHaveAttribute('aria-disabled', 'true')
    }
  }

  /** Types the words into the field of the open palette. */
  async writeInField(text: string): Promise<void> {
    this.scenario.noteAction()
    const field = this.palette.field()
    await expect(field).toBeVisible()
    await field.fill(text)
  }

  /** The open palette lists the command under its name as many times as the count says. */
  async expectCommandCount(label: string, count: number): Promise<void> {
    await expect(this.palette.rowNamed(new RegExp('^' + escapeRegExp(label) + '$'))).toHaveCount(count)
  }

  private async readAsk(): Promise<Ask | undefined> {
    if (await this.palette.treeField().isVisible()) return { placeholder: 'Name', groups: [], isInTree: true }
    const field = this.palette.field()
    if (!(await field.isVisible())) return undefined
    const placeholder = (await field.getAttribute('placeholder')) ?? ''
    if (placeholder === COMMAND_PLACEHOLDER) return undefined
    return { placeholder, groups: await this.palette.readGroupTitles(), isInTree: false }
  }

  private async waitForAsk(key: string): Promise<Ask> {
    let ask: Ask | undefined
    try {
      await expect
        .poll(
          async () => {
            ask = await this.readAsk()
            return ask !== undefined && isAskedFor(key, ask.placeholder, ask.groups)
          },
          { timeout: ASK_LIMIT_MS },
        )
        .toBe(true)
    } catch {
      const asked = ask === undefined ? 'nothing' : `"${ask.placeholder}" (lists: ${ask.groups.map((one) => `"${one}"`).join(', ') || 'none'})`
      throw new Error(`the row "${key}" does not match what the palette asks for: ${asked}`)
    }
    return ask as Ask
  }

  private async give(ask: Ask, key: string, value: string): Promise<void> {
    const { page } = this.scenario
    if (ask.isInTree) {
      const field = this.palette.treeField()
      await field.fill(value)
      await field.press('Enter')
      await expect(field).toBeHidden()
      return
    }
    const field = this.palette.field()
    if (ask.placeholder === CONFIRM_PLACEHOLDER) {
      await this.pickRow(value, false)
      return
    }
    const isSearch = getPaletteField(ask.placeholder) !== 'name' && getPaletteField(ask.placeholder) !== 'text'
    if (isSearch) {
      await field.fill(value)
      await this.pickRow(value, true)
      return
    }
    if (key === 'Name' && this.scenario.hasWorld === false) await this.scenario.claimTitle(value)
    await field.fill(value)
    await page.keyboard.press('Enter')
  }

  private async pickRow(value: string, isExact: boolean): Promise<void> {
    const pattern = new RegExp(`^${escapeRegExp(value)}${isExact ? '$' : ''}`)
    const rows = this.palette.rowNamed(pattern)
    await expect(rows.first(), `the palette lists no row named "${value}"`).toBeVisible()
    if ((await rows.count()) > 1) throw new Error(`more than one row of the palette is named "${value}": write more of the name`)
    await rows.first().click()
  }

  private async expectNothingAsked(): Promise<void> {
    await expect
      .poll(async () => this.readAsk(), { timeout: ASK_LIMIT_MS })
      .toBeUndefined()
      .catch(async () => {
        const ask = await this.readAsk()
        throw new Error(`the palette still asks for "${ask?.placeholder ?? ''}": add a row for it`)
      })
  }
}
