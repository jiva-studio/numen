import type { PluginSettings, Result } from '@numen/plugin-sdk'
import type { SubscriptionClient } from '../api/subscription'
import { useSubscriptionOffers } from './subscriptionOffers'
import { useSubscriptionPayment } from './subscriptionPayment'
import { useSubscriptionSession } from './subscriptionSession'
import { useSubscriptionSummary } from './subscriptionSummary'

export interface UseSubscriptionOptions {
  readonly client?: SubscriptionClient | undefined
  readonly openExternal: (url: string) => Promise<Result<null>>
  readonly now?: (() => Date) | undefined
  readonly pollEveryMs?: number | undefined
  readonly pollLimitMs?: number | undefined
  readonly refreshMs?: Readonly<Record<'noIdentity' | 'ready' | 'unreachable', number>> | undefined
  readonly settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
}

/** The subscription card: the account as the server tells it, the plans it offers, the purchase and what the card says. */
export function useSubscription(options: UseSubscriptionOptions) {
  const session = useSubscriptionSession(options)
  const payment = useSubscriptionPayment(options, session)
  const offers = useSubscriptionOffers(session.account)
  const summary = useSubscriptionSummary(session.account, session.server)
  return { session, payment, offers, summary }
}
