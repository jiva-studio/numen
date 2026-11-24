import type { PluginContext, PluginSettings } from '@numen/plugin-sdk'
import type { AccountClient } from '../api/account'
import type { SubscriptionClient } from '../api/subscription'
import { useAccountData, type SaveFile } from './accountData'
import { useSubscriptionOffers } from './subscriptionOffers'
import { useSubscriptionPayment } from './subscriptionPayment'
import { useSubscriptionSession } from './subscriptionSession'
import { useSubscriptionSummary } from './subscriptionSummary'

export interface UseSubscriptionOptions {
  readonly client?: SubscriptionClient | undefined
  readonly accountClient?: AccountClient | undefined
  readonly saveFile?: SaveFile | undefined
  readonly openExternal: PluginContext['identity']['openExternal']
  readonly now?: (() => Date) | undefined
  readonly pollEveryMs?: number | undefined
  readonly pollLimitMs?: number | undefined
  readonly refreshMs?: Readonly<Record<'noIdentity' | 'ready' | 'unreachable', number>> | undefined
  readonly settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
}

/**
 * The subscription card: the account as the server tells it, the plans it offers, the purchase and what the card says.
 * With an account client it also holds the deletion of the cloud data; while the account is being deleted the server
 * stops answering the subscription, so the card stops asking.
 */
export function useSubscription(options: UseSubscriptionOptions) {
  const accountData =
    options.accountClient === undefined || options.saveFile === undefined
      ? undefined
      : useAccountData({ client: options.accountClient, saveFile: options.saveFile })
  const session = useSubscriptionSession({
    ...options,
    isPaused: () => accountData?.state.isDeleting.value === true,
  })
  const payment = useSubscriptionPayment(options, session)
  const offers = useSubscriptionOffers(session.account)
  const summary = useSubscriptionSummary(session.account, session.server)

  async function start(): Promise<void> {
    if (accountData !== undefined) await accountData.actions.load()
    await session.load()
  }

  return { session, payment, offers, summary, accountData, start }
}
