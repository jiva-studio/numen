/** The person at the card of the subscription on the page "Identity and licenses": buys a plan, changes it and reads what the card says. */
import { expect, type Locator, type Page } from 'playwright/test'
import type { Scenario } from './scenario.js'
import type { SettingsDriver } from './settings.js'

const PAGE = 'Identity and licenses'
const SUBSCRIPTION_TITLE = 'Numen Sync'
const CHOOSE = /^Choose /
const CHANGE_PLAN = 'Change plan'
const MORE_ACTIONS = 'More actions'
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

/** The plan row of the card: the size in GB and the price, with the name in front. */
export const getPlanLinePattern = (name: string, gigabytes: number): RegExp => new RegExp(`^${name}, ${gigabytes} GB(,|$)`)

/** The name of the item of the menu "Change plan" of a plan: the name of the plan, then its size and price or nothing. */
export const getPlanItemPattern = (name: string): RegExp => new RegExp(`^${name}(?![A-Za-z])`)

/** The day as the card of the subscription writes it, such as "December 7, 2026". */
export const formatDay = (day: Date): string => day.toLocaleDateString('en-US', { day: 'numeric', month: 'long', year: 'numeric' })

export class SubscriptionDriver {
  constructor(
    private readonly scenario: Scenario,
    private readonly settings: SettingsDriver,
  ) {}

  private async showPage(): Promise<Locator> {
    return this.settings.showPage(PAGE)
  }

  private getCard(panel: Locator): Locator {
    return panel.getByRole('region', { name: SUBSCRIPTION_TITLE, exact: true })
  }

  /** Expects the card to show the plan of a name and a size in GB: the name in front in the plan row. */
  async expectPlan(name: string, gigabytes: number): Promise<void> {
    const card = this.getCard(await this.showPage())
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
    const more = card.getByRole('button', { name: MORE_ACTIONS, exact: true })
    await expect(more).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
    await more.click()
    await this.scenario.page.getByRole('menuitem', { name: CHANGE_PLAN, exact: true }).click()
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
    await expect(card.getByText(new RegExp(`^${name}, `)).first()).toBeVisible({ timeout: SUBSCRIPTION_LIMIT_MS })
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
