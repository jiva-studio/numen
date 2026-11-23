/** The person at the page "Identity and licenses": sets up an identity, restores it from its words and deletes it. */
import { expect, type Locator, type Page } from 'playwright/test'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const CONFIRMATIONS = 3
const ACCOUNT_FINGERPRINT = 'Account fingerprint'
const NO_IDENTITY = 'No identity on this device'
const SUBSCRIPTION_TITLE = 'Numen Sync'
const BUY = 'Buy'
const PAYMENT_RECEIVED = 'Payment received'
const PAYMENT_ROUTE = /\/pay\//
/** The plugin reads the subscription when the card opens and every 3 seconds while a payment is awaited. */
const SUBSCRIPTION_LIMIT_MS = 20_000

/** Matches the button that buys the plan of a name with its size in GB; the card may add the price after the size. */
export const getBuyName = (name: string, gigabytes: number): RegExp => new RegExp(`^Buy ${name}, ${gigabytes} GB(?:,|$)`)

/** Matches the button that buys the plan of a name, whatever its size. */
export const getBuyPattern = (name: string): RegExp => new RegExp(`^Buy ${name}, `)

/** The name of the button that changes the plan to the plan of a name. */
export const getChangeName = (name: string): string => `Change plan to ${name}`

/** The text of the plan line of the card: the name and the size in GB. */
export const getPlanLine = (name: string, gigabytes: number): string => `${name}, ${gigabytes} GB`


const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']

/** The day as the card of the subscription writes it, such as "7 Dec 2026". */
export const formatDay = (day: Date): string => `${day.getDate()} ${MONTHS[day.getMonth()] ?? ''} ${day.getFullYear()}`

/** Matches the button that buys the product of a name; the card may add the price after the name. */
export const getBuyProductName = (name: string): RegExp => new RegExp(`^Buy ${name}(?:,|$)`)

/** The place a field of the confirmation asks for: "Confirm word 4" asks for the 4th word. */
export const readPosition = (label: string): number => {
  const found = /^Confirm word (\d+)$/.exec(label)
  if (found?.[1] === undefined) throw new Error(`"${label}" is not the label of a confirmation of a word`)
  return Number(found[1])
}

/**
 * The words to type for the places the page asks about. A mistake replaces the first word that is not the mistake already,
 * so the answer is wrong whatever the words are.
 */
export const planAnswers = (positions: readonly number[], words: readonly string[], mistake?: string): string[] => {
  const answers = positions.map((position) => {
    const word = words.at(position - 1)
    if (word === undefined) throw new Error(`the page asks for the word ${position} of ${words.length}`)
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
    const value = panel.locator('dt', { hasText: ACCOUNT_FINGERPRINT }).locator('xpath=following-sibling::dd[1]')
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

  async expectWordCount(count: number): Promise<void> {
    await expect(this.getShownWords(await this.showPage())).toHaveCount(count)
  }

  async expectNoWords(): Promise<void> {
    const panel = await this.showPage()
    await expect(panel.getByRole('list', { name: 'Recovery words' })).toHaveCount(0)
    for (const word of this.words) await expect(panel.getByText(word, { exact: true })).toHaveCount(0)
  }

  private async confirm(mistake?: string): Promise<void> {
    const panel = await this.showPage()
    const fields = panel.getByRole('textbox', { name: /^Confirm word \d+$/ })
    await expect(fields).toHaveCount(CONFIRMATIONS)
    const labels = await fields.evaluateAll((all) => all.map((one) => one.getAttribute('aria-label') ?? ''))
    const answers = planAnswers(labels.map(readPosition), this.words, mistake)
    for (const [index, answer] of answers.entries()) await fields.nth(index).fill(answer)
    await panel.getByRole('button', { name: 'Keep identity', exact: true }).click()
  }

  async confirmWords(count: number): Promise<void> {
    if (count !== CONFIRMATIONS) throw new Error(`the page asks for ${CONFIRMATIONS} words, not ${count}`)
    await this.confirm()
  }

  async confirmWithMistake(word: string): Promise<void> {
    await this.confirm(word)
  }

  async create(): Promise<void> {
    await this.start()
    await this.confirm()
    this.account = await this.readAccount(await this.showPage())
  }

  async expectIdentity(has: boolean): Promise<void> {
    const panel = await this.showPage()
    if (has) await expect(panel.locator('dt', { hasText: ACCOUNT_FINGERPRINT })).toBeVisible()
    else await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  async delete(): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Delete from this device', exact: true }).click()
    await this.scenario.page.getByRole('alertdialog').getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(panel.getByText(NO_IDENTITY)).toBeVisible()
  }

  private async restore(text: string): Promise<void> {
    const panel = await this.showPage()
    await panel.getByRole('button', { name: 'Restore from words', exact: true }).click()
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
    await expect((await this.showPage()).getByRole('alert')).toContainText(text)
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
    return panel.locator('.settings-card').filter({ hasText: SUBSCRIPTION_TITLE })
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
    const buy = card.getByRole('button', { name: BUY, exact: true })
    if (isOffered) await expect(buy).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    else {
      await expect(card.getByText(SUBSCRIPTION_TITLE).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
      await expect(buy).toHaveCount(0)
    }
  }

  private async openPayment(): Promise<Page> {
    const card = this.getCard(await this.showPage())
    const opened = this.scenario.page.waitForEvent('popup')
    await card.getByRole('button', { name: BUY, exact: true }).click()
    return opened
  }

  /** Pays on the page of the stand-in and waits until the card shows the subscription as active. */
  async buy(): Promise<void> {
    const payment = await this.openPayment()
    await expect(payment.getByText(PAYMENT_RECEIVED)).toBeVisible()
    await payment.close()
    await this.expectSubscription('Active')
  }

  /** Buys the plan of a name on the page of the stand-in and waits until the card shows the subscription as active. */
  async buyPlan(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    const opened = this.scenario.page.waitForEvent('popup')
    await card.getByRole('button', { name: getBuyPattern(name) }).click()
    const payment = await opened
    await expect(payment.getByText(PAYMENT_RECEIVED)).toBeVisible()
    await payment.close()
    await this.expectSubscription('Active')
  }

  /** Expects the card to offer to buy the plan of a name with its size in GB. */
  async expectBuyOffer(name: string, gigabytes: number): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(card.getByRole('button', { name: getBuyName(name, gigabytes) })).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  private getProductOwnedMark(card: Locator, name: string): Locator {
    return card.getByRole('list', { name: 'Products' }).getByRole('listitem').filter({ hasText: name }).getByText('Purchased', { exact: true })
  }

  /** Buys the product of a name on the page of the stand-in and waits until the card shows it as owned. */
  async buyProduct(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    const opened = this.scenario.page.waitForEvent('popup')
    await card.getByRole('button', { name: getBuyProductName(name) }).click()
    const payment = await opened
    await expect(payment.getByText(PAYMENT_RECEIVED)).toBeVisible()
    await payment.close()
    await this.expectProductOwned(name)
  }

  /** Expects the card to offer to buy the product of a name. */
  async expectProductOffer(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(card.getByRole('button', { name: getBuyProductName(name) })).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Expects the card to show the product of a name as owned and to offer no purchase of it. */
  async expectNoProductOffer(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(this.getProductOwnedMark(card, name)).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await expect(card.getByRole('button', { name: getBuyProductName(name) })).toHaveCount(0)
  }

  /** Expects the card to show the product of a name as owned. */
  async expectProductOwned(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(this.getProductOwnedMark(card, name)).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
  }

  /** Expects the card to offer to buy no plan, after it showed the subscription. */
  async expectNoBuyOffer(): Promise<void> {
    const card = this.getCard(await this.showPage())
    await expect(card.getByText(SUBSCRIPTION_TITLE).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await expect(card.getByRole('button', { name: /^Buy / })).toHaveCount(0)
  }

  /** Expects the card to offer, or not to offer, to change the plan to the plan of a name. */
  async expectChangeOffer(name: string, isOffered: boolean): Promise<void> {
    const card = this.getCard(await this.showPage())
    const change = card.getByRole('button', { name: getChangeName(name), exact: true })
    if (isOffered) await expect(change).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    else {
      await expect(card.getByText(SUBSCRIPTION_TITLE).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
      await expect(change).toHaveCount(0)
    }
  }

  /** Changes the plan on the card and waits until the card shows the plan of the name. */
  async changePlan(name: string): Promise<void> {
    const card = this.getCard(await this.showPage())
    await card.getByRole('button', { name: getChangeName(name), exact: true }).click()
    await expect(card.getByText(new RegExp(`^${name}, `)).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
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
