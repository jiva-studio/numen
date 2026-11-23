import { onScopeDispose } from 'vue'
import type { Result } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { getFailureCode, type SubscriptionClient } from '../api/subscription'
import { subscriptionClient } from '../api/wire'
import { WORDS as words } from '../words'
import { getMessage } from './subscriptionAccount'
import type { useSubscriptionSession } from './subscriptionSession'

const DEFAULT_POLL_EVERY_MS = 3000
const DEFAULT_POLL_LIMIT_MS = 600_000

export interface SubscriptionPaymentOptions {
  readonly client?: SubscriptionClient | undefined
  readonly openExternal: (url: string) => Promise<Result<null>>
  readonly pollEveryMs?: number | undefined
  readonly pollLimitMs?: number | undefined
}

/**
 * The purchase of a plan: it opens the payment page in the browser, then asks the server until the
 * subscription is active or the limit passes.
 */
export function useSubscriptionPayment(
  options: SubscriptionPaymentOptions,
  session: ReturnType<typeof useSubscriptionSession>,
) {
  const {
    client = subscriptionClient,
    openExternal,
    pollEveryMs = DEFAULT_POLL_EVERY_MS,
    pollLimitMs = DEFAULT_POLL_LIMIT_MS,
  } = options
  const { account, waiting, problem, load, readIfReachable } = session
  let pollTimer: ReturnType<typeof setTimeout> | undefined
  let limitTimer: ReturnType<typeof setTimeout> | undefined

  const isPaid = (): boolean => account.phase.value === 'active'

  function reportProblem(failure: unknown): void {
    problem.value = getMessage(failure)
  }

  function endPaymentPoll(): void {
    clearTimeout(pollTimer)
    clearTimeout(limitTimer)
    pollTimer = undefined
    limitTimer = undefined
    waiting.value = false
  }

  async function poll(): Promise<void> {
    await readIfReachable()
    if (!waiting.value) return
    if (isPaid()) {
      endPaymentPoll()
      return
    }
    pollTimer = setTimeout(() => runInBackground(poll(), reportProblem), pollEveryMs)
  }

  function beginPaymentPoll(): void {
    waiting.value = true
    pollTimer = setTimeout(() => runInBackground(poll(), reportProblem), pollEveryMs)
    limitTimer = setTimeout(() => {
      endPaymentPoll()
      problem.value = words.paymentNotSeen
    }, pollLimitMs)
  }

  async function buy(offer: string): Promise<void> {
    if (waiting.value) return
    problem.value = ''
    try {
      const opened = await openExternal(await client.createCheckout(offer))
      if (opened.ok) beginPaymentPoll()
      else problem.value = opened.error
    } catch (failure) {
      if (getFailureCode(failure) === 'already_exists') await load()
      else problem.value = getMessage(failure)
    }
  }

  onScopeDispose(endPaymentPoll)

  return { waiting, buy }
}
