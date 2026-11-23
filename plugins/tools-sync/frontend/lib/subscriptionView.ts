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

/** A plan the person can buy or move to: its id and the label of the button, such as "Plus, 100 GB". */
export interface PlanOffer {
  readonly id: string
  readonly name: string
  readonly label: string
}

/** What the card shows. isTrial is true while a trial runs that no payment has followed; the end of the period is then the end of the trial. */
export interface SubscriptionSnapshot {
  readonly status: SubscriptionStatus
  readonly phase: SubscriptionPhase | ''
  readonly dates: SubscriptionDates
  readonly isTrial?: boolean | undefined
  /** The address of the sync server in use; empty when it is not known. */
  readonly server?: string | undefined
}

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
const NUMERIC_DATE = new Intl.DateTimeFormat('en-GB', {
  day: 'numeric',
  month: 'numeric',
  year: 'numeric',
})

function formatDate(time: string): string {
  const at = Date.parse(time)
  if (Number.isNaN(at)) return ''
  const parts = new Map(NUMERIC_DATE.formatToParts(at).map((one) => [one.type, one.value]))
  const month = MONTHS.at(Number(parts.get('month')) - 1) ?? ''
  return `${Number(parts.get('day'))} ${month} ${parts.get('year') ?? ''}`
}

const getPhaseView = (
  phase: SubscriptionPhase,
  dates: SubscriptionDates,
  isTrial: boolean,
): SubscriptionView => {
  const periodEnd = formatDate(dates.periodEnd)
  switch (phase) {
    case 'none':
      return { badge: words.noSubscription, text: words.noSubscriptionText, tone: 'neutral' }
    case 'active':
      return {
        badge: words.active,
        text: isTrial ? words.trialText(periodEnd) : words.activeText(periodEnd),
        tone: 'positive',
      }
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

/** The badge and the text of the card for what the server told. */
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
      return { badge: words.noIdentityBadge, text: words.noIdentityText, tone: 'neutral' }
    case 'ready':
      return snapshot.phase === ''
        ? { badge: '', text: '', tone: 'neutral' }
        : getPhaseView(snapshot.phase, snapshot.dates, snapshot.isTrial === true)
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

/** The label of an offer: the name, the size and the price, such as "Plus, 100 GB, $15". */
export function getOfferLabel(plan: PlanSize): string {
  const price = formatPrice(plan.priceCents, plan.currency)
  return price === '' ? getPlanLabel(plan) : `${getPlanLabel(plan)}, ${price}`
}

/** The space in use against the quota, such as "2 GB of 100 GB used". It may be above the quota. */
export function getUsageText(usedBytes: number, quotaBytes: number): string {
  return `${formatSize(usedBytes)} of ${formatSize(quotaBytes)} used`
}
