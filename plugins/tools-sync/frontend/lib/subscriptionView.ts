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

export interface SubscriptionSnapshot {
  readonly status: SubscriptionStatus
  readonly phase: SubscriptionPhase | ''
  readonly dates: SubscriptionDates
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
  return `${parts.get('day') ?? ''} ${month} ${parts.get('year') ?? ''}`
}

const getPhaseView = (phase: SubscriptionPhase, dates: SubscriptionDates): SubscriptionView => {
  const periodEnd = formatDate(dates.periodEnd)
  switch (phase) {
    case 'none':
      return { badge: words.noSubscription, text: words.noSubscriptionText, tone: 'neutral' }
    case 'active':
      return { badge: words.active, text: words.activeText(periodEnd), tone: 'positive' }
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
      return { badge: words.syncServerUnreachable, text: '', tone: 'warning' }
    case 'noIdentity':
      return { badge: words.noIdentityBadge, text: words.noIdentityText, tone: 'neutral' }
    case 'ready':
      return snapshot.phase === ''
        ? { badge: '', text: '', tone: 'neutral' }
        : getPhaseView(snapshot.phase, snapshot.dates)
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

/** The size in GB, where 1 GB is 2^30 bytes, such as "10 GB" or "1.5 GB". */
export function formatSize(_bytes: number): string {
  throw new Error('not implemented')
}

/** The price for one period, such as "$5", "$4.99", "€10" or "7 CHF"; empty when the plan has no price. */
export function formatPrice(_cents: number, _currency: string): string {
  throw new Error('not implemented')
}

/** The name and the size of a plan, such as "Plus, 100 GB"; only the size when the plan has no name. */
export function getPlanLabel(_plan: PlanSize): string {
  throw new Error('not implemented')
}

/** The label of an offer: the name, the size and the price, such as "Plus, 100 GB, $15". */
export function getOfferLabel(_plan: PlanSize): string {
  throw new Error('not implemented')
}

/** The space in use against the quota, such as "2 GB of 100 GB used". It may be above the quota. */
export function getUsageText(_usedBytes: number, _quotaBytes: number): string {
  throw new Error('not implemented')
}
