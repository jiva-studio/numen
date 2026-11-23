import { WORDS as words } from '../words'

export type SubscriptionPhase = 'none' | 'active' | 'grace' | 'readOnly' | 'deleted'

export type SubscriptionStatus = 'loading' | 'ready' | 'unreachable' | 'noIdentity'

/** What the card says: the badge and the text under the title. */
export interface SubscriptionView {
  readonly badge: string
  readonly text: string
  readonly tone: 'neutral' | 'positive' | 'warning'
}

/** The times of the subscription as the server sent them, RFC 3339 or empty. */
export interface SubscriptionDates {
  readonly periodEnd: string
  readonly writesEnd: string
  readonly deleteAt: string
}

/** A plan the person can buy or move to: its name, its size such as "100 GB" and its price such as "$15 / month", empty for a plan with no price. */
export interface PlanOffer {
  readonly id: string
  readonly name: string
  readonly size: string
  readonly price: string
}

/** What the card shows; the server address and the plan name are empty when the client does not know them. */
interface SubscriptionSnapshot {
  readonly status: SubscriptionStatus
  readonly phase: SubscriptionPhase | ''
  readonly dates: SubscriptionDates
  readonly server?: string | undefined
  readonly planName?: string | undefined
}

const LONG_DATE = new Intl.DateTimeFormat('en-US', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

function formatDate(time: string): string {
  const at = Date.parse(time)
  return Number.isNaN(at) ? '' : LONG_DATE.format(at)
}

const getPhaseView = (
  phase: SubscriptionPhase,
  dates: SubscriptionDates,
  planName: string,
): SubscriptionView => {
  const periodEnd = formatDate(dates.periodEnd)
  switch (phase) {
    case 'none':
      return { badge: '', text: words.noSubscriptionText, tone: 'neutral' }
    case 'active':
      return { badge: planName, text: '', tone: 'positive' }
    case 'grace':
      return {
        badge: words.ended,
        text: words.graceText(periodEnd, formatDate(dates.writesEnd)),
        tone: 'warning',
      }
    case 'readOnly':
      return {
        badge: words.readOnly,
        text: words.readOnlyText(periodEnd, formatDate(dates.deleteAt)),
        tone: 'warning',
      }
    case 'deleted':
      return { badge: words.deleted, text: words.deletedText, tone: 'warning' }
  }
}

/**
 * The badge and the text of the card for what the server told. A badge appears only where it adds to the
 * text and to the plans: the plan of an active subscription, and the state of an ended one.
 */
export function getSubscriptionView(snapshot: SubscriptionSnapshot): SubscriptionView {
  switch (snapshot.status) {
    case 'loading':
      return { badge: '', text: words.subscriptionLoading, tone: 'neutral' }
    case 'unreachable':
      return {
        badge: words.syncServerUnreachable,
        text: words.unreachableText(snapshot.server ?? ''),
        tone: 'warning',
      }
    case 'noIdentity':
      return { badge: '', text: words.noIdentityText, tone: 'neutral' }
    case 'ready':
      return snapshot.phase === ''
        ? { badge: '', text: '', tone: 'neutral' }
        : getPhaseView(snapshot.phase, snapshot.dates, snapshot.planName ?? '')
  }
}

/** The line of an active subscription that says how long it is paid for, such as "Active until" and "November 23, 2026". */
export interface Validity {
  readonly label: string
  readonly date: string
}

/**
 * The end of the paid period, such as "November 23, 2026"; it is the end of the trial while a trial runs that
 * no payment has followed, and the date is empty when the time is empty or unreadable.
 */
export function getValidity(dates: SubscriptionDates, isTrial: boolean): Validity {
  return {
    label: isTrial ? words.trialUntil : words.activeUntil,
    date: formatDate(dates.periodEnd),
  }
}

/** A plan to show, as the server lists it. */
export interface PlanSize {
  readonly id: string
  readonly name: string
  readonly quotaBytes: number
  readonly priceCents: number
  readonly currency: string
}

const BYTES_IN_GB = 2 ** 30
const CENTS_IN_UNIT = 100
const CURRENCY_SYMBOLS: Readonly<Record<string, string>> = { USD: '$', EUR: '€', GBP: '£' }

/** The size in GB, where 1 GB is 2^30 bytes, such as "10 GB" or "1.5 GB". */
export function formatSize(bytes: number): string {
  return `${Number((bytes / BYTES_IN_GB).toFixed(1))} GB`
}

function formatPrice(cents: number, currency: string): string {
  if (cents === 0) return ''
  const units = cents / CENTS_IN_UNIT
  const amount = Number.isInteger(units) ? String(units) : units.toFixed(2)
  const symbol = CURRENCY_SYMBOLS[currency]
  return symbol === undefined ? `${amount} ${currency}` : `${symbol}${amount}`
}

/** The name and the size of a plan, such as "Plus, 100 GB"; only the size when the plan has no name. */
export function getPlanLabel(plan: PlanSize): string {
  const size = formatSize(plan.quotaBytes)
  return plan.name === '' ? size : `${plan.name}, ${size}`
}

/** The price of a plan for a month, such as "$15 / month"; empty for a plan with no price. */
export function getPlanPrice(plan: PlanSize): string {
  const price = formatPrice(plan.priceCents, plan.currency)
  return price === '' ? '' : words.pricePerMonth(price)
}

/** The space in use against the quota, such as "2 GB of 100 GB used". It may be above the quota. */
export function getUsageText(usedBytes: number, quotaBytes: number): string {
  return `${formatSize(usedBytes)} of ${formatSize(quotaBytes)} used`
}

/** The share of the quota in use as a whole percent from 0 to 100; 100 where the space in use is above the quota. */
export function getUsagePercent(usedBytes: number, quotaBytes: number): number {
  if (quotaBytes <= 0) return 0
  return Math.min(100, Math.round((usedBytes / quotaBytes) * 100))
}
