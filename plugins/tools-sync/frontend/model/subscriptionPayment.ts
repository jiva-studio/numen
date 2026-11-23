import { onScopeDispose, ref } from 'vue'
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
 * subscription is active, the limit passes or the person stops waiting.
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
  const starting = ref(false)
  let attempt = 0
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

  async function openCheckout(offer: string, isCurrent: () => boolean): Promise<void> {
    const url = await client.createCheckout(offer)
    if (!isCurrent()) return
    const opened = await openExternal(url)
    if (!isCurrent()) return
    if (opened.ok) beginPaymentPoll()
    else problem.value = opened.error
  }

  async function buy(offer: string): Promise<void> {
    if (waiting.value || starting.value) return
    problem.value = ''
    starting.value = true
    attempt += 1
    const current = attempt
    const isCurrent = (): boolean => current === attempt
    try {
      await openCheckout(offer, isCurrent)
    } catch (failure) {
      if (!isCurrent()) return
      if (getFailureCode(failure) === 'already_exists') await load()
      else problem.value = getMessage(failure)
    } finally {
      if (isCurrent()) starting.value = false
    }
  }

  /** Stops waiting for the payment: the payment page stays closed when it is not open yet, and the server is asked no more. */
  function cancel(): void {
    attempt += 1
    starting.value = false
    endPaymentPoll()
  }

  onScopeDispose(endPaymentPoll)

  return { waiting, starting, buy, cancel }
}
