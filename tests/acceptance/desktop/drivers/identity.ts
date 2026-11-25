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
import { getSeedFingerprint, getSeedWords, type IdentityComputer } from './identity-seed.js'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
/** A word of the list that makes no identity when it is written again and again. */
const WRONG_WORD = 'abandon'
/** A dialog that finishes its work closes within this time; one that is still open then waits for the person. */
const DIALOG_CLOSES_MS = 3_000
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
  private isSeeded = false

  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
  ) {}

  /** Shows the page; a dialog that is open is closed first, as a person leaves it. */
  private async showPage(): Promise<Locator> {
    const dialog = this.scenario.page.getByRole('dialog')
    const isClosed = await expect(dialog).toHaveCount(0, { timeout: DIALOG_CLOSES_MS }).then(() => true, () => false)
    if (!isClosed) {
      await dialog.getByRole('button', { name: 'Cancel', exact: true }).click()
      await expect(dialog).toHaveCount(0)
    }
    return this.settings.showPage(PAGE)
  }

  /** The dialogs of the identity are in the body of the window, not in the panel of the page, and block the page while they are open. */
  private getShownWords(): Locator {
    return this.scenario.page.getByRole('list', { name: 'Recovery words' }).getByRole('listitem')
  }

  private async readAccount(panel: Locator): Promise<string> {
    const value = panel
      .locator('.settings__row', { hasText: ACCOUNT_FINGERPRINT })
      .locator('.settings__value')
    await expect(value).toBeVisible()
    return (await value.innerText()).trim()
  }

  /** Gives this computer the keys of the test identities: the application finds them at its start and treats the person as one who set the identity up. */
  give(computer: IdentityComputer): void {
    this.scenario.setIdentity(computer)
    this.words = getSeedWords()
    this.account = getSeedFingerprint()
    this.isSeeded = true
  }

  /** Whether the identity of the person came from the test identities. */
  hasGivenIdentity(): boolean {
    return this.isSeeded
  }

  async start(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Create identity', exact: true }).click()
    const shown = this.getShownWords()
    await expect(shown.first()).toBeVisible()
    this.words = (await shown.allInnerTexts()).map((word) => word.trim())
  }

  async writeWordsDown(): Promise<void> {
    await this.scenario.page.getByRole('button', { name: 'I wrote the words down', exact: true }).click()
  }

  async expectWordsAsked(count: number): Promise<void> {
    const { page } = this.scenario
    await expect(page.getByRole('textbox', { name: /^Word \d+$/ })).toHaveCount(count)
    await expect(page.getByRole('list', { name: 'Recovery words' })).toHaveCount(0)
  }

  async expectWordCount(count: number): Promise<void> {
    await expect(this.getShownWords()).toHaveCount(count)
  }

  async expectNoWords(): Promise<void> {
    await this.showPage()
    const { page } = this.scenario
    await expect(page.getByRole('list', { name: 'Recovery words' })).toHaveCount(0)
    for (const word of this.words) await expect(page.getByText(word, { exact: true })).toHaveCount(0)
  }

  private async confirm(mistake?: string): Promise<void> {
    const { page } = this.scenario
    const fields = page.getByRole('textbox', { name: /^Word \d+$/ })
    await expect(fields).toHaveCount(CONFIRMATIONS)
    const labels = await fields.evaluateAll((all) =>
      all.map((one) => (one as HTMLInputElement).labels?.[0]?.textContent.trim() ?? ''),
    )
    const answers = planAnswers(labels.map(readPosition), this.words, mistake)
    for (const [index, answer] of answers.entries()) await fields.nth(index).fill(answer)
    await page.getByRole('button', { name: 'Continue', exact: true }).click()
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

  private getKitDialog(): Locator {
    return this.scenario.page.getByRole('dialog', { name: 'Download recovery kit' })
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
    const dialog = this.getKitDialog()
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

  private getRestoreDialog(title: string): Locator {
    return this.scenario.page.getByRole('dialog', { name: title })
  }

  private async restore(text: string): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    await panel.page().getByRole('menuitem', { name: 'Type the 12 words' }).click()
    const dialog = this.getRestoreDialog('Recovery words')
    await dialog.getByRole('textbox', { name: PROOF_FIELD, exact: true }).fill(text)
    const restore = dialog.getByRole('button', { name: 'Restore', exact: true })
    if (await restore.isEnabled()) await restore.click()
  }

  private async restoreSheets(texts: readonly string[]): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    await panel.page().getByRole('menuitem', { name: 'Type the words of two sheets' }).click()
    const dialog = this.getRestoreDialog('Recovery kit: type')
    for (const [index, text] of texts.entries()) {
      await dialog.getByRole('textbox', { name: `Sheet ${index + 1}`, exact: true }).fill(text)
    }
    const restore = dialog.getByRole('button', { name: 'Restore', exact: true })
    if (await restore.isEnabled()) await restore.click()
  }

  async restoreFromRecoveryWords(): Promise<void> {
    await this.restore(this.words.join(' '))
  }

  async restoreFromWords(text: string): Promise<void> {
    await this.restore(text)
  }

  /** Restores from a number of words that are not the words of an identity. */
  async restoreFromWrongWords(count: number): Promise<void> {
    await this.restore(Array.from({ length: count }, () => WRONG_WORD).join(' '))
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
