/**
 * The person at the page "Identity and licenses": sets up an identity, prints and checks its recovery kit,
 * restores it from its words or from two sheets and deletes it.
 */
import { expect, type Locator } from 'playwright/test'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const CONFIRMATIONS = 3
const ACCOUNT_FINGERPRINT = 'Account fingerprint'
const NO_IDENTITY = 'An identity is a key kept only on your devices.'
const SHEETS_IN_KIT = 3
const WORDS_ON_SHEET = 33
const KIT_BUTTON = /^(Skip for now|Done)$/
const MISTAKE_AT = 5
const OTHER_ACCOUNTS: readonly (readonly string[])[] = [
  'letter advice cage absurd amount doctor acoustic avoid letter advice cage above'.split(' '),
  'legal winner thank year wave sausage worth useful legal winner thank yellow'.split(' '),
]

/** The numbers of the sheets a step names: "2" is the sheet 2, "1 and 3" are the sheets 1 and 3 in that order. */
export const readSheetNumbers = (text: string): number[] => {
  const parts = text.split(/\s+and\s+/)
  const numbers = parts.map((part) => (/^\d+$/.test(part.trim()) ? Number(part) : NaN))
  if (numbers.some((one) => !Number.isInteger(one) || one < 1 || one > SHEETS_IN_KIT)) {
    throw new Error(`"${text}" does not name sheets of a kit of ${SHEETS_IN_KIT}`)
  }
  return numbers
}

const changeWord = (word: string): string => (word === 'academic' ? 'acid' : 'academic')

/** The text to type for each named sheet. The sheet with a mistake has one word replaced by another word of the list. */
export const planSheetTexts = (
  sheets: readonly (readonly string[])[],
  numbers: readonly number[],
  mistakeSheet?: number,
): string[] => {
  if (mistakeSheet !== undefined && !numbers.includes(mistakeSheet)) {
    throw new Error(`the sheet ${mistakeSheet} is not among the sheets ${numbers.join(' and ')}`)
  }
  return numbers.map((number) => {
    const words = sheets.at(number - 1)
    if (words === undefined) throw new Error(`the kit has no sheet ${number}`)
    if (number !== mistakeSheet) return words.join(' ')
    return words.map((word, at) => (at === MISTAKE_AT ? changeWord(word) : word)).join(' ')
  })
}

/** Valid recovery words of an account other than the one of `own`. */
export const pickOtherWords = (own: readonly string[]): readonly string[] => {
  const other = OTHER_ACCOUNTS.find((words) => words.join(' ') !== own.join(' '))
  if (other === undefined) throw new Error('no other account is left to pick')
  return other
}

declare global {
  interface Window {
    __numenPrintCalls?: number
  }
}

const readPrintCount = (): number => window.__numenPrintCalls ?? 0

const spyOnPrint = (): void => {
  if (typeof window.__numenPrintCalls === 'number') return
  window.__numenPrintCalls = 0
  window.print = () => {
    window.__numenPrintCalls = (window.__numenPrintCalls ?? 0) + 1
  }
}

/** The place a field of the confirmation asks for: "Word 4" asks for the 4th word. */
export const readPosition = (label: string): number => {
  const found = /^Word (\d+)$/.exec(label)
  if (found?.[1] === undefined)
    throw new Error(`"${label}" is not the label of a confirmation of a word`)
  return Number(found[1])
}

/**
 * The words to type for the places the page asks about. A mistake replaces the first word that is not the mistake already,
 * so the answer is wrong whatever the words are.
 */
export const planAnswers = (
  positions: readonly number[],
  words: readonly string[],
  mistake?: string,
): string[] => {
  const answers = positions.map((position) => {
    const word = words.at(position - 1)
    if (word === undefined)
      throw new Error(`the page asks for the word ${position} of ${words.length}`)
    return word
  })
  if (mistake === undefined) return answers
  const at = answers.findIndex((word) => word !== mistake)
  if (at === -1) throw new Error(`every word asked for is "${mistake}": nothing can take its place`)
  return answers.map((word, index) => (index === at ? mistake : word))
}

export class IdentityDriver {
  private words: readonly string[] = []
  private account = ''
  private sheets: readonly (readonly string[])[] = []
  private printed = 0

  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
  ) {}

  private async showPage(): Promise<Locator> {
    return this.settings.showPage(PAGE)
  }

  private getShownWords(panel: Locator): Locator {
    return panel.getByRole('list', { name: 'Recovery words' }).getByRole('listitem')
  }

  private async readAccount(panel: Locator): Promise<string> {
    const value = panel
      .locator('.settings__row', { hasText: ACCOUNT_FINGERPRINT })
      .locator('.settings__value')
    await expect(value).toBeVisible()
    return (await value.innerText()).trim()
  }

  async start(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Create identity', exact: true }).click()
    const shown = this.getShownWords(panel)
    await expect(shown.first()).toBeVisible()
    this.words = (await shown.allInnerTexts()).map((word) => word.trim())
  }

  async writeWordsDown(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'I wrote the words down', exact: true }).click()
  }

  async expectWordsAsked(count: number): Promise<void> {
    const panel = await this.showPage()
    await expect(panel.getByRole('textbox', { name: /^Word \d+$/ })).toHaveCount(count)
    await expect(panel.getByRole('list', { name: 'Recovery words' })).toHaveCount(0)
  }

  async expectWordCount(count: number): Promise<void> {
    await expect(this.getShownWords(await this.showPage())).toHaveCount(count)
  }

  async expectNoWords(): Promise<void> {
    const panel = await this.showPage()
    await expect(panel.getByRole('list', { name: 'Recovery words' })).toHaveCount(0)
    for (const word of this.words)
      await expect(panel.getByText(word, { exact: true })).toHaveCount(0)
  }

  private async confirm(mistake?: string): Promise<void> {
    const panel = await this.showPage()
    const fields = panel.getByRole('textbox', { name: /^Word \d+$/ })
    await expect(fields).toHaveCount(CONFIRMATIONS)
    const labels = await fields.evaluateAll((all) =>
      all.map((one) => (one as HTMLInputElement).labels?.[0]?.textContent?.trim() ?? ''),
    )
    const answers = planAnswers(labels.map(readPosition), this.words, mistake)
    for (const [index, answer] of answers.entries()) await fields.nth(index).fill(answer)
    await panel.getByRole('button', { name: 'Continue', exact: true }).click()
  }

  async confirmWords(count: number): Promise<void> {
    if (count !== CONFIRMATIONS)
      throw new Error(`the page asks for ${CONFIRMATIONS} words, not ${count}`)
    await this.confirm()
  }

  async confirmWithMistake(word: string): Promise<void> {
    await this.confirm(word)
  }

  async create(): Promise<void> {
    await this.start()
    await this.writeWordsDown()
    await this.confirm()
    const panel = await this.showPage()
    const leave = panel.getByRole('button', { name: KIT_BUTTON })
    await expect(leave).toBeVisible()
    await leave.click()
    this.account = await this.readAccount(panel)
  }

  private async closeKit(panel: Locator): Promise<void> {
    const leave = panel.getByRole('button', { name: KIT_BUTTON })
    if ((await leave.count()) > 0) await leave.click()
  }

  private async installPrintSpy(): Promise<void> {
    await this.scenario.page.addInitScript(spyOnPrint)
    await this.scenario.page.evaluate(spyOnPrint)
  }

  private async makeKit(words: readonly string[]): Promise<Locator> {
    const panel = await this.showPage()
    await this.closeKit(panel)
    await panel.getByRole('button', { name: 'Print recovery kit', exact: true }).click()
    await panel.getByRole('textbox', { name: 'Recovery words', exact: true }).fill(words.join(' '))
    await panel.getByRole('button', { name: 'Make kit', exact: true }).click()
    return panel
  }

  private getSheet(panel: Locator, number: number): Locator {
    return panel.getByLabel(`Sheet ${number} of ${SHEETS_IN_KIT}`, { exact: true })
  }

  async print(): Promise<void> {
    await this.installPrintSpy()
    const panel = await this.makeKit(this.words)
    const sheets: string[][] = []
    for (let number = 1; number <= SHEETS_IN_KIT; number += 1) {
      const items = this.getSheet(panel, number).getByRole('listitem')
      await expect(items.first()).toBeVisible()
      sheets.push((await items.allInnerTexts()).map((word) => word.trim()))
    }
    this.sheets = sheets
    await panel.getByRole('button', { name: 'Print', exact: true }).click()
    this.printed += 1
  }

  async printWithOtherWords(): Promise<void> {
    await this.makeKit(pickOtherWords(this.words))
  }

  async expectSheetCount(count: number): Promise<void> {
    const panel = await this.showPage()
    await expect(panel.getByRole('article')).toHaveCount(count)
    for (let number = 1; number <= count; number += 1) {
      await expect(this.getSheet(panel, number).getByRole('listitem')).toHaveCount(WORDS_ON_SHEET)
    }
  }

  async expectPrinted(): Promise<void> {
    expect(this.printed, 'no kit was sent to the printer by the person').toBeGreaterThan(0)
    await expect.poll(() => this.scenario.page.evaluate(readPrintCount)).toBe(this.printed)
  }

  private async typeSheets(
    panel: Locator,
    numbers: readonly number[],
    mistakeSheet?: number,
  ): Promise<void> {
    const texts = planSheetTexts(this.sheets, numbers, mistakeSheet)
    for (const [index, text] of texts.entries()) {
      await panel.getByRole('textbox', { name: `Sheet ${index + 1}`, exact: true }).fill(text)
    }
  }

  async checkSheets(numbers: string, mistakeSheet?: number): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Check two sheets', exact: true }).first().click()
    await this.typeSheets(panel, readSheetNumbers(numbers), mistakeSheet)
    await panel.getByRole('button', { name: 'Check', exact: true }).click()
  }

  async restoreFromSheets(numbers: string, mistakeSheet?: number): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore from sheets', exact: true }).click()
    await this.typeSheets(panel, readSheetNumbers(numbers), mistakeSheet)
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
  }

  async expectSays(text: string): Promise<void> {
    await expect((await this.showPage()).getByRole('status')).toContainText(text, {
      ignoreCase: true,
    })
  }

  async expectIdentity(has: boolean): Promise<void> {
    const panel = await this.showPage()
    if (has)
      await expect(panel.locator('.settings__row', { hasText: ACCOUNT_FINGERPRINT })).toBeVisible()
    else await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  async delete(): Promise<void> {
    const panel = await this.showPage()
    await this.closeKit(panel)
    await panel.getByRole('button', { name: 'Delete identity', exact: true }).click()
    await this.scenario.page
      .getByRole('alertdialog')
      .getByRole('button', { name: 'Delete', exact: true })
      .click()
    await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  private async restore(text: string): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    await panel.getByRole('textbox', { name: 'Your 12 recovery words' }).fill(text)
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
  }

  async restoreFromRecoveryWords(): Promise<void> {
    await this.restore(this.words.join(' '))
  }

  async restoreFromWords(text: string): Promise<void> {
    await this.restore(text)
  }

  async expectWarning(text: string): Promise<void> {
    await expect((await this.showPage()).getByRole('alert')).toContainText(text, {
      ignoreCase: true,
    })
  }

  async expectSameAccount(): Promise<void> {
    expect(this.account, 'no identity was created before').not.toBe('')
    await expect.poll(async () => this.readAccount(await this.showPage())).toBe(this.account)
  }
}
