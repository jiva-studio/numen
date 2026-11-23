import type { SubscriptionPhase } from '../api/subscription'
import { WORDS as words } from '../words'

export type SubscriptionStatus = 'loading' | 'ready' | 'unreachable' | 'noIdentity'

/** How loud the badge of the card is. */
export type BadgeTone = 'neutral' | 'positive' | 'warning'

/** What the card says: the badge and the text under the title. */
export interface SubscriptionView {
  readonly badge: string
  readonly text: string
  readonly tone: BadgeTone
}

/** The times of the subscription as the server sent them, RFC 3339 or empty. */
export interface SubscriptionDates {
  readonly periodEnd: string
  readonly writesEnd: string
  readonly deleteAt: string
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

/** A local date such as `2 Jan 2006`, and an empty text for a time that is not there. */
export function formatDate(time: string): string {
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
