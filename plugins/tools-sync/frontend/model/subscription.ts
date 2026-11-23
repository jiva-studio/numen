import { computed, onScopeDispose, ref } from 'vue'
import type { PluginSettings, Result } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import {
  getFailureCode,
  type PlanResponse,
  type SubscriptionClient,
  type SubscriptionResponse,
} from '../api/subscription'
import { subscriptionClient } from '../api/wire'
import {
  formatSize,
  getOfferLabel,
  getPlanLabel,
  getSubscriptionView,
  getUsageText,
  type SubscriptionPhase,
  type SubscriptionDates,
  type PlanOffer,
  type SubscriptionStatus,
} from '../lib/subscriptionView'
import { WORDS as words } from '../words'

const DEFAULT_POLL_EVERY_MS = 3000
const DEFAULT_POLL_LIMIT_MS = 600_000
const REFRESH_MS = { noIdentity: 3000, ready: 10_000, unreachable: 30_000 } as const
const SERVER_SETTING = 'server'
const NO_DATES: SubscriptionDates = { periodEnd: '', writesEnd: '', deleteAt: '' }

export interface UseSubscriptionOptions {
  readonly client?: SubscriptionClient | undefined
  readonly openExternal: (url: string) => Promise<Result<null>>
  readonly now?: (() => Date) | undefined
  readonly pollEveryMs?: number | undefined
  readonly pollLimitMs?: number | undefined
  /** How long the card waits before it asks again, by what it saw last. */
  readonly refreshMs?: Readonly<Record<'noIdentity' | 'ready' | 'unreachable', number>> | undefined
  /** Where the address of the sync server is read and followed. */
  readonly settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
}

const readServer = (settings: UseSubscriptionOptions['settings']): string =>
  settings?.get<string>(SERVER_SETTING) ?? ''

const getMessage = (failure: unknown): string =>
  failure instanceof Error ? failure.message : String(failure)

export function useSubscription(options: UseSubscriptionOptions) {
  const {
    client = subscriptionClient,
    openExternal,
    pollEveryMs = DEFAULT_POLL_EVERY_MS,
    pollLimitMs = DEFAULT_POLL_LIMIT_MS,
    refreshMs = REFRESH_MS,
    settings,
  } = options
  const server = ref(readServer(settings))
  const status = ref<SubscriptionStatus>('loading')
  const phase = ref<SubscriptionPhase | ''>('')
  const dates = ref<SubscriptionDates>(NO_DATES)
  const plan = ref('')
  const quotaBytes = ref(0)
  const usedBytes = ref(0)
  const isTrial = ref(false)
  const plans = ref<readonly PlanResponse[]>([])
  const waiting = ref(false)
  const problem = ref('')
  let isAwaited: () => boolean = () => false
  let pollTimer: ReturnType<typeof setTimeout> | undefined
  let limitTimer: ReturnType<typeof setTimeout> | undefined
  let refreshTimer: ReturnType<typeof setTimeout> | undefined
  let isDisposed = false

  const canBuy = computed(() => status.value === 'ready' && phase.value !== 'active')
  const view = computed(() =>
    getSubscriptionView({
      status: status.value,
      phase: phase.value,
      dates: dates.value,
      isTrial: isTrial.value,
      server: server.value,
    }),
  )

  function apply(answer: SubscriptionResponse): void {
    status.value = answer.hasIdentity ? 'ready' : 'noIdentity'
    phase.value = answer.hasIdentity ? answer.phase : ''
    dates.value = answer.hasIdentity
      ? { periodEnd: answer.periodEndAt, writesEnd: answer.writesEndAt, deleteAt: answer.deleteAt }
      : NO_DATES
    plan.value = answer.hasIdentity ? answer.plan : ''
    quotaBytes.value = answer.hasIdentity ? answer.quotaBytes : 0
    usedBytes.value = answer.hasIdentity ? answer.usedBytes : 0
    isTrial.value = answer.hasIdentity && answer.isTrial === true
  }

  async function read(): Promise<void> {
    apply(await client.getSubscription())
  }

  async function loadPlans(): Promise<void> {
    try {
      plans.value = await client.listPlans()
    } catch (failure) {
      plans.value = []
      problem.value = getMessage(failure)
    }
  }

  async function refresh(): Promise<void> {
    const before = status.value
    try {
      await read()
    } catch (failure) {
      status.value = 'unreachable'
      if (getFailureCode(failure) !== 'unavailable') problem.value = getMessage(failure)
      return
    }
    if (before === 'unreachable') problem.value = ''
    if (status.value === 'ready' && (before !== 'ready' || plans.value.length === 0)) {
      await loadPlans()
    }
  }

  function scheduleRefresh(): void {
    clearTimeout(refreshTimer)
    refreshTimer = undefined
    if (isDisposed) return
    const wait =
      status.value === 'ready'
        ? refreshMs.ready
        : refreshMs[status.value === 'noIdentity' ? 'noIdentity' : 'unreachable']
    refreshTimer = setTimeout(() => runInBackground(tick(), reportProblem), wait)
  }

  async function tick(): Promise<void> {
    if (!waiting.value) await refresh()
    scheduleRefresh()
  }

  /** Reads the subscription and keeps reading it, so the card follows the identity, the server and the payment. */
  async function load(): Promise<void> {
    try {
      await refresh()
    } finally {
      scheduleRefresh()
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
    if (isAwaited()) {
      endPaymentPoll()
      return
    }
    pollTimer = setTimeout(() => runInBackground(poll(), reportProblem), pollEveryMs)
  }

  function beginPaymentPoll(isPaid: () => boolean): void {
    isAwaited = isPaid
    waiting.value = true
    pollTimer = setTimeout(() => runInBackground(poll(), reportProblem), pollEveryMs)
    limitTimer = setTimeout(() => {
      endPaymentPoll()
      problem.value = words.paymentNotSeen
    }, pollLimitMs)
  }

  async function pay(offer: string, isPaid: () => boolean): Promise<void> {
    if (waiting.value) return
    problem.value = ''
    try {
      const opened = await openExternal(await client.createCheckout(offer))
      if (opened.ok) beginPaymentPoll(isPaid)
      else problem.value = opened.error
    } catch (failure) {
      if (getFailureCode(failure) === 'already_exists') await load()
      else problem.value = getMessage(failure)
    }
  }

  const buy = (offer: string): Promise<void> => pay(offer, () => phase.value === 'active')

  async function change(offer: string): Promise<void> {
    problem.value = ''
    try {
      apply(await client.changePlan(offer))
    } catch (failure) {
      const code = getFailureCode(failure)
      if (code === 'already_exists' || code === 'failed_precondition') await readIfReachable()
      else problem.value = getMessage(failure)
    }
  }

  async function manage(): Promise<void> {
    problem.value = ''
    try {
      const address = await client.createPortalSession?.()
      if (address === undefined) return
      const opened = await openExternal(address)
      if (!opened.ok) problem.value = opened.error
    } catch (failure) {
      problem.value = getMessage(failure)
    }
  }

  const followServer = settings?.onChange<string>(SERVER_SETTING, (address) => {
    server.value = address
    runInBackground(load(), reportProblem)
  })

  onScopeDispose(() => {
    isDisposed = true
    clearTimeout(refreshTimer)
    followServer?.dispose()
    endPaymentPoll()
  })

  const toOffer = (one: PlanResponse): PlanOffer => ({
    id: one.id,
    name: one.name,
    label: getOfferLabel(one),
  })
  const buyOffers = computed<readonly PlanOffer[]>(() =>
    canBuy.value ? plans.value.map(toOffer) : [],
  )
  const changeOffers = computed<readonly PlanOffer[]>(() =>
    status.value === 'ready' && phase.value === 'active'
      ? plans.value.filter((one) => one.id !== plan.value).map(toOffer)
      : [],
  )
  const planLine = computed(() => {
    if (plan.value === '') return ''
    const known = plans.value.find((one) => one.id === plan.value)
    return known === undefined
      ? formatSize(quotaBytes.value)
      : getPlanLabel({ ...known, quotaBytes: quotaBytes.value })
  })
  const hasUsage = computed(
    () =>
      status.value === 'ready' &&
      quotaBytes.value > 0 &&
      (phase.value === 'active' || phase.value === 'grace'),
  )
  const usage = computed(() =>
    hasUsage.value ? getUsageText(usedBytes.value, quotaBytes.value) : '',
  )
  const isFull = computed(() => hasUsage.value && usedBytes.value >= quotaBytes.value)
  const canManage = computed(
    () => status.value === 'ready' && phase.value !== 'none' && phase.value !== '',
  )

  return {
    server,
    status,
    phase,
    canBuy,
    canManage,
    waiting,
    problem,
    view,
    buyOffers,
    changeOffers,
    planLine,
    usage,
    isFull,
    load,
    buy,
    change,
    manage,
  }
}
