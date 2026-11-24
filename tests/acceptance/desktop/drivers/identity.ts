/**
 * The person at the page "Identity and licenses": sets up an identity, downloads its recovery kit as
 * a PDF, restores it from its words or from two sheets of the kit and deletes it.
 */
import fs from 'node:fs'
import path from 'node:path'
import zlib from 'node:zlib'
import { expect, type Locator, type Page } from 'playwright/test'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const CONFIRMATIONS = 3
const ACCOUNT_FINGERPRINT = 'Account fingerprint'
const NO_IDENTITY = 'An identity is your Numen account.'
const SUBSCRIPTION_TITLE = 'Numen Sync'
const CHOOSE = /^Choose /
const CHANGE_PLAN = 'Change plan'
const CURRENT_PLAN = 'Current plan'
const SWITCH_QUESTION = /^Switch to .+\?$/
const CANCEL = 'Cancel'
const RENEWS = 'Renews'
const PAYMENT_RECEIVED = 'Payment received'
const PAYMENT_ROUTE = /\/pay\//
/** The plugin reads the subscription when the card opens and every 3 seconds while a payment is awaited. */
const SUBSCRIPTION_LIMIT_MS = 20_000

/** The name of the row of the card that chooses the plan of a name, such as "Choose Plus, 100 GB, $15 / month"; with the size in GB when it is given. */
export const getChoosePattern = (name: string, gigabytes?: number): RegExp => new RegExp(`^Choose ${name}, ${gigabytes === undefined ? '' : `${gigabytes} GB(,|$)`}`)

/** The plan row of the card: the size in GB and the price, with the name in front once the subscription ended. */
export const getPlanLinePattern = (name: string, gigabytes: number): RegExp => new RegExp(`^(${name}, )?${gigabytes} GB(,|$)`)

/** The name of the item of the menu "Change plan" of a plan: the name of the plan, then its size and price or nothing. */
export const getPlanItemPattern = (name: string): RegExp => new RegExp(`^${name}(?![A-Za-z])`)

/** The day as the card of the subscription writes it, such as "December 7, 2026". */
export const formatDay = (day: Date): string => day.toLocaleDateString('en-US', { day: 'numeric', month: 'long', year: 'numeric' })
const SHEETS_IN_KIT = 3
const WORDS_ON_SHEET = 33
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

const KIT_FILE = 'Numen recovery kit.pdf'

/**
 * The text each page of a PDF shows, in the order it is drawn: the strings of the text operators of
 * each content stream, inflated when it is compressed.
 */
export const readPdfPages = (pdf: Buffer): string[][] => {
  const pages: string[][] = []
  const text = pdf.toString('latin1')
  for (const found of text.matchAll(/stream\r?\n([\s\S]*?)\r?\nendstream/g)) {
    const raw = Buffer.from(found[1] ?? '', 'latin1')
    let content: string
    try {
      content = zlib.inflateSync(raw).toString('latin1')
    } catch {
      content = raw.toString('latin1')
    }
    const shown = [...content.matchAll(/\(((?:[^()\\]|\\.)*)\) Tj/g)].map((one) => one[1] ?? '')
    if (shown.length > 0) pages.push(shown)
  }
  return pages
}

/** The words of each sheet of a kit: the strings of a page that are one lower-case word, in order. */
export const readKitSheets = (pages: readonly (readonly string[])[]): string[][] =>
  pages.map((page) => page.filter((one) => /^[a-z]+$/.test(one)))

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
      .getByRole('button', { name: 'Download PDF', exact: true })
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
    await dialog
      .getByRole('textbox', { name: 'Recovery word 1', exact: true })
      .fill(words.join(' '))
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

  private async typeSheets(
    dialog: Locator,
    numbers: readonly number[],
    mistakeSheet?: number,
  ): Promise<void> {
    const texts = planSheetTexts(this.sheets, numbers, mistakeSheet)
    for (const [index, text] of texts.entries()) {
      await dialog.getByRole('textbox', { name: `Sheet ${index + 1}`, exact: true }).fill(text)
    }
  }

  async restoreFromSheets(numbers: string, mistakeSheet?: number): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    const dialog = panel.getByRole('dialog', { name: 'Restore identity' })
    await dialog.getByRole('radio', { name: 'Recovery kit', exact: true }).click()
    await this.typeSheets(dialog, readSheetNumbers(numbers), mistakeSheet)
    await dialog.getByRole('button', { name: 'Restore', exact: true }).click()
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

  private async restore(text: string): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore', exact: true }).click()
    const dialog = panel.getByRole('dialog', { name: 'Restore identity' })
    await dialog.getByRole('textbox', { name: 'Recovery word 1', exact: true }).fill(text)
    await dialog.getByRole('button', { name: 'Restore', exact: true }).click()
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

  /** The account fingerprint of the last identity this person created. */
  getAccountFingerprint(): string {
    expect(this.account, 'no identity was created before').not.toBe('')
    return this.account
  }

  async expectSameAccount(): Promise<void> {
    expect(this.account, 'no identity was created before').not.toBe('')
    await expect.poll(async () => this.readAccount(await this.showPage())).toBe(this.account)
  }

  private getCard(panel: Locator): Locator {
    return panel.getByRole('region', { name: SUBSCRIPTION_TITLE, exact: true })
  }

  private getBadge(card: Locator, text: string): Locator {
    return card.locator('header').getByText(text, { exact: true })
  }

  /** Expects the card to show the plan of a name and a size in GB: the name in the badge of an active subscription, in the plan row of an ended one. */
  async expectPlan(name: string, gigabytes: number): Promise<void> {
    const card = this.getCard(await this.showPage())
    const named = this.getBadge(card, name).or(card.getByText(new RegExp(`^${name}, ${gigabytes} GB`)))
    await expect(named.first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await expect(card.getByText(getPlanLinePattern(name, gigabytes)).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Expects the card of the subscription to show the badge or the text; the plugin asks the server every 3 seconds while a payment is awaited. */
  async expectSubscription(text: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(card.getByText(text, { exact: true }).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Expects the page to have the text, in the card of the subscription or beside it. */
  async expectPageSays(text: string): Promise<void> {
    await expect((await this.showPage()).getByText(text).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  async expectOffer(isOffered: boolean): Promise<void> {
    const card = this.getCard(await this.showPage())
    const buy = card.getByRole('button', { name: CHOOSE })
    if (isOffered) await expect(buy.first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    else {
      await expect(card.getByText(SUBSCRIPTION_TITLE).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
      await expect(buy).toHaveCount(0)
    }
  }

  private async openPayment(): Promise<Page> {
    const card = this.getCard(await this.showPage())
    const opened = this.scenario.page.waitForEvent('popup')
    await card.getByRole('button', { name: CHOOSE }).first().click()
    return opened
  }

  /** Pays on the page of the stand-in and waits until the card shows the subscription as active. */
  async startPurchase(): Promise<void> {
    const payment = await this.openPayment()
    await expect(payment.getByText(PAYMENT_RECEIVED)).toBeVisible()
    await payment.close()
    await this.expectActive()
  }

  /** Expects the card to show an active subscription: the row with the day it renews. */
  async expectActive(): Promise<void> {
    await this.expectSubscription(RENEWS)
  }

  /** Expects the card to say until which day the trial runs. */
  async expectTrialUntil(day: Date): Promise<void> {
    await this.expectSubscription('Trial until')
    await this.expectSubscription(formatDay(day))
  }

  /** Buys the plan of a name on the page of the stand-in and waits until the card shows the subscription as active. */
  async choosePlan(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    const opened = this.scenario.page.waitForEvent('popup')
    await card.getByRole('button', { name: getChoosePattern(name) }).click()
    const payment = await opened
    await expect(payment.getByText(PAYMENT_RECEIVED)).toBeVisible()
    await payment.close()
    await this.expectActive()
  }

  /** Expects the card to offer to choose the plan of a name on a row with its size in GB. */
  async expectBuyOffer(name: string, gigabytes: number): Promise<void> {
    const card = this.getCard(await this.showPage())
    const row = card.getByRole('button', { name: getChoosePattern(name, gigabytes) }).filter({ hasText: `${gigabytes} GB of synced files` })
    await expect(row).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Expects the card to offer to buy no plan, after it showed the subscription. */
  async expectNoBuyOffer(): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(card.getByText(SUBSCRIPTION_TITLE).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await expect(card.getByRole('button', { name: CHOOSE })).toHaveCount(0)
  }

  private async openChangeItem(card: Locator, name: string): Promise<Locator> {
    const change = card.getByRole('button', { name: CHANGE_PLAN, exact: true })
    await expect(change).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await change.click()
    return this.scenario.page.getByRole('menuitemradio', { name: getPlanItemPattern(name) })
  }

  /** Expects the menu "Change plan" of the card to offer, or not to offer, the plan of a name: the current plan is listed and cannot be chosen. */
  async expectChangeOffer(name: string, isOffered: boolean): Promise<void> {
    const card = this.getCard(await this.showPage())
    const item = await this.openChangeItem(card, name)
    if (isOffered) await expect(item).toBeEnabled()
    else await expect(item.and(this.scenario.page.locator(':enabled'))).toHaveCount(0)
    await this.scenario.page.keyboard.press('Escape')
  }

  /** Expects the menu "Change plan" of the card to check the plan of a name as the current plan. */
  async expectCurrentPlan(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    const item = await this.openChangeItem(card, name)
    await expect(item).toHaveAttribute('aria-checked', 'true')
    await expect(item).toContainText(CURRENT_PLAN)
    await this.scenario.page.keyboard.press('Escape')
  }

  /** Chooses the plan of a name in the menu "Change plan" of the card, which asks to confirm the switch. */
  async chooseChange(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await (await this.openChangeItem(card, name)).click()
    await expect(this.getSwitchQuestion(name)).toBeVisible()
  }

  private getSwitchQuestion(name: string): Locator {
    return this.scenario.page.getByRole('alertdialog', { name: `Switch to ${name}?`, exact: true })
  }

  /** Confirms the switch to the plan of a name and waits until the card shows that plan. */
  async confirmChange(name: string): Promise<void> {
    const question = this.getSwitchQuestion(name)
    await question.getByRole('button', { name: `Switch to ${name}`, exact: true }).click()
    await expect(question).toHaveCount(0)
    const card = this.getCard(await this.showPage())
    await expect(this.getBadge(card, name)).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Cancels the switch the card asks about. */
  async cancelChange(): Promise<void> {
    const question = this.scenario.page.getByRole('alertdialog', { name: SWITCH_QUESTION })
    await question.getByRole('button', { name: CANCEL, exact: true }).click()
    await expect(question).toHaveCount(0)
  }

  /** Changes the plan from the menu of the card, confirms the switch and waits until the card shows the plan of the name. */
  async changePlan(name: string): Promise<void> {
    await this.chooseChange(name)
    await this.confirmChange(name)
  }

  /** Opens the page of the payment and stops its load before the stand-in is asked, so nothing is paid. */
  async openPaymentWithoutPaying(): Promise<void> {
    const context = this.scenario.page.context()
    await context.route(PAYMENT_ROUTE, (route) => route.abort())
    try {
      const payment = await this.openPayment()
      await payment.close()
    } finally {
      await context.unroute(PAYMENT_ROUTE)
    }
  }
}
