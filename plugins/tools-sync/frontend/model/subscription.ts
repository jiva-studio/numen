import { computed, onScopeDispose, ref } from 'vue'
import type { Failure, OpenExternalErrorCode, Result } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { getFailureCode, type SubscriptionClient } from '../api/subscription'
import { subscriptionClient } from '../api/wire'
import {
  getSubscriptionView,
  type SubscriptionPhase,
  type SubscriptionDates,
  type SubscriptionStatus,
} from '../lib/subscriptionView'
import { WORDS as words } from '../words'

const DEFAULT_POLL_EVERY_MS = 3000
const DEFAULT_POLL_LIMIT_MS = 600_000
const NO_DATES: SubscriptionDates = { periodEnd: '', writesEnd: '', deleteAt: '' }

interface UseSubscriptionOptions {
  readonly client?: SubscriptionClient | undefined
  readonly openExternal: (url: string) => Promise<Result<null, Failure<OpenExternalErrorCode>>>
  readonly now?: (() => Date) | undefined
  readonly pollEveryMs?: number | undefined
  readonly pollLimitMs?: number | undefined
}

const getMessage = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

export function useSubscription(options: UseSubscriptionOptions) {
  const {
    client = subscriptionClient,
    openExternal,
    pollEveryMs = DEFAULT_POLL_EVERY_MS,
    pollLimitMs = DEFAULT_POLL_LIMIT_MS,
  } = options
  const status = ref<SubscriptionStatus>('loading')
  const phase = ref<SubscriptionPhase | ''>('')
  const dates = ref<SubscriptionDates>(NO_DATES)
  const waiting = ref(false)
  const problem = ref('')
  let pollTimer: ReturnType<typeof setTimeout> | undefined
  let limitTimer: ReturnType<typeof setTimeout> | undefined

  const canBuy = computed(() => status.value === 'ready' && phase.value !== 'active')
  const view = computed(() =>
    getSubscriptionView({ status: status.value, phase: phase.value, dates: dates.value }),
  )

  async function read(): Promise<void> {
    const answer = await client.getSubscription()
    status.value = answer.hasIdentity ? 'ready' : 'noIdentity'
    phase.value = answer.hasIdentity ? answer.phase : ''
    dates.value = answer.hasIdentity
      ? { periodEnd: answer.periodEndAt, writesEnd: answer.writesEndAt, deleteAt: answer.deleteAt }
      : NO_DATES
  }

  async function load(): Promise<void> {
    try {
      await read()
    } catch (failure) {
      status.value = 'unreachable'
      if (getFailureCode(failure) !== 'unavailable') problem.value = getMessage(failure)
    }
  }

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

  async function readIfReachable(): Promise<boolean> {
    try {
      await read()
      return true
    } catch {
      return false
    }
  }

  async function poll(): Promise<void> {
    await readIfReachable()
    if (!waiting.value) return
    if (phase.value === 'active') {
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

  async function startPurchase(): Promise<void> {
    if (waiting.value) return
    problem.value = ''
    try {
      const opened = await openExternal(await client.createCheckout())
      if (opened.ok) beginPaymentPoll()
      else problem.value = words.checkoutNotOpened[opened.error.error]
    } catch (failure) {
      if (getFailureCode(failure) === 'already_exists') await load()
      else problem.value = getMessage(failure)
    }
  }

  onScopeDispose(endPaymentPoll)

  return { status, phase, canBuy, waiting, problem, view, load, startPurchase }
}
