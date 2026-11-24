/**
 * The person at the page "Identity and licenses": sets up an identity, downloads its recovery kit as
 * a PDF, restores it from its words or from two sheets of the kit and deletes it.
 */
import fs from 'node:fs'
import path from 'node:path'
import { expect, type Locator } from 'playwright/test'
import {
  KIT_FILE,
  planAnswers,
  pickOtherWords,
  planSheetTexts,
  readKitSheets,
  readPdfPages,
  readPosition,
  readSheetNumbers,
  SHEETS_IN_KIT,
  WORDS_ON_SHEET,
} from './identity-kit.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const CONFIRMATIONS = 3
const ACCOUNT_FINGERPRINT = 'Identity fingerprint'
const NO_IDENTITY = 'Your identity in Numen lives on your own computer'
const PROOF_FIELD = 'Recovery words or sheets'

export class IdentityDriver {
  private words: readonly string[] = []
  private account = ''
  private sheets: readonly (readonly string[])[] = []
  private downloads = 0
  private kitFile = ''

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
      all.map((one) => (one as HTMLInputElement).labels?.[0]?.textContent.trim() ?? ''),
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
    this.account = await this.readAccount(await this.showPage())
  }

  private prepareDownload(): string {
    const folder = path.join(this.scenario.getWorld().root, `downloads-${++this.downloads}`)
    fs.mkdirSync(folder, { recursive: true })
    this.scenario.getServer().writeFolderChoice(folder)
    return path.join(folder, KIT_FILE)
  }

  private async readDownload(file: string): Promise<void> {
    await expect
      .poll(() => fs.existsSync(file), { message: `no kit was saved at ${file}` })
      .toBe(true)
    this.kitFile = file
    this.sheets = readKitSheets(readPdfPages(fs.readFileSync(file)))
  }

  private getKitDialog(panel: Locator): Locator {
    return panel.getByRole('dialog', { name: 'Download recovery kit' })
  }

  private getDownloadButton(panel: Locator): Locator {
    return panel
      .locator('.settings__row', { hasText: 'Recovery kit' })
      .getByRole('button', { name: 'Download recovery kit', exact: true })
  }

  async download(): Promise<void> {
    const file = this.prepareDownload()
    const panel = await this.showPage()
    await this.getDownloadButton(panel).click()
    await this.readDownload(file)
  }

  private async downloadWith(words: readonly string[]): Promise<string> {
    const file = this.prepareDownload()
    const panel = await this.showPage()
    await this.getDownloadButton(panel).click()
    const dialog = this.getKitDialog(panel)
    await dialog.getByRole('textbox', { name: PROOF_FIELD, exact: true }).fill(words.join(' '))
    await dialog.getByRole('button', { name: 'Download PDF', exact: true }).click()
    return file
  }

  async downloadWithRecoveryWords(): Promise<void> {
    await this.readDownload(await this.downloadWith(this.words))
  }

  async downloadWithOtherWords(): Promise<void> {
    await this.downloadWith(pickOtherWords(this.words))
  }

  expectKitSaved(count: number): void {
    expect(this.kitFile, 'no kit was downloaded').not.toBe('')
    const pdf = fs.readFileSync(this.kitFile)
    expect(pdf.subarray(0, 5).toString('latin1')).toBe('%PDF-')
    expect(this.sheets).toHaveLength(count)
    for (const sheet of this.sheets) expect(sheet).toHaveLength(WORDS_ON_SHEET)
  }

  expectKitShowsAccount(): void {
    expect(this.account, 'no identity was created before').not.toBe('')
    const pages = readPdfPages(fs.readFileSync(this.kitFile))
    expect(pages).toHaveLength(SHEETS_IN_KIT)
    for (const page of pages) expect(page).toContain(this.account)
  }

  async restoreFromSheets(numbers: string, mistakeSheet?: number): Promise<void> {
    const texts = planSheetTexts(this.sheets, readSheetNumbers(numbers), mistakeSheet)
    await this.restoreSheets(texts)
  }

  async expectIdentity(has: boolean): Promise<void> {
    const panel = await this.showPage()
    if (has)
      await expect(panel.locator('.settings__row', { hasText: ACCOUNT_FINGERPRINT })).toBeVisible()
    else await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  async delete(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Delete identity', exact: true }).click()
    await this.scenario.page
      .getByRole('alertdialog')
      .getByRole('button', { name: 'Delete', exact: true })
      .click()
    await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  private getRestoreDialog(panel: Locator, title: string): Locator {
    return panel.getByRole('dialog', { name: title })
  }

  private async restore(text: string): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    await panel.page().getByRole('menuitem', { name: 'Type the 12 words' }).click()
    const dialog = this.getRestoreDialog(panel, 'Recovery words')
    await dialog.getByRole('textbox', { name: PROOF_FIELD, exact: true }).fill(text)
    const restore = dialog.getByRole('button', { name: 'Restore', exact: true })
    if (await restore.isEnabled()) await restore.click()
  }

  private async restoreSheets(texts: readonly string[]): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    await panel.page().getByRole('menuitem', { name: 'Type the words of two sheets' }).click()
    const dialog = this.getRestoreDialog(panel, 'Recovery kit: type')
    for (const [index, text] of texts.entries()) {
      await dialog.getByRole('textbox', { name: `Sheet ${index + 1}`, exact: true }).fill(text)
    }
    const restore = dialog.getByRole('button', { name: 'Restore', exact: true })
    if (await restore.isEnabled()) await restore.click()
  }

  async expectRestoreNote(text: string): Promise<void> {
    const dialog = (await this.showPage()).getByRole('dialog', { name: /^(Recovery words|Recovery kit: type)$/ })
    await expect(dialog.getByRole('status').or(dialog.getByRole('alert'))).toContainText(text, {
      ignoreCase: true,
    })
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

  /** The recovery words of the last identity this person started. */
  getWords(): readonly string[] {
    expect(this.words.length, 'no identity was started before').toBeGreaterThan(0)
    return this.words
  }

  /** The identity fingerprint of the last identity this person created. */
  getAccountFingerprint(): string {
    expect(this.account, 'no identity was created before').not.toBe('')
    return this.account
  }

  async expectSameAccount(): Promise<void> {
    expect(this.account, 'no identity was created before').not.toBe('')
    await expect.poll(async () => this.readAccount(await this.showPage())).toBe(this.account)
  }
}
