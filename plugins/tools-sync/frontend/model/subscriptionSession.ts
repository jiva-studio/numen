import { onScopeDispose, ref } from 'vue'
import type { PluginSettings, Result } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { getFailureCode, type SubscriptionClient } from '../api/subscription'
import { subscriptionClient } from '../api/wire'
import { applyAnswer, createAccount, getMessage } from './subscriptionAccount'

const REFRESH_MS = { noIdentity: 3000, ready: 10_000, unreachable: 30_000 } as const
const SERVER_SETTING = 'server'

/** refreshMs is the wait before the next read by the last status; settings hold the server address the card reads and follows. */
export interface SubscriptionSessionOptions {
  readonly client?: SubscriptionClient | undefined
  readonly openExternal: (url: string) => Promise<Result<null>>
  readonly refreshMs?: Readonly<Record<'noIdentity' | 'ready' | 'unreachable', number>> | undefined
  readonly settings?: Pick<PluginSettings, 'get' | 'onChange'> | undefined
}

/**
 * The account of the card as the server tells it: it reads it, keeps reading it so the card follows the
 * identity, the server and the payment, and changes the plan or opens the portal.
 */
export function useSubscriptionSession(options: SubscriptionSessionOptions) {
  const { client = subscriptionClient, openExternal, refreshMs = REFRESH_MS, settings } = options
  const account = createAccount()
  const server = ref(settings?.get<string>(SERVER_SETTING) ?? '')
  const waiting = ref(false)
  const problem = ref('')
  let refreshTimer: ReturnType<typeof setTimeout> | undefined
  let isDisposed = false

  async function read(): Promise<void> {
    applyAnswer(account, await client.getSubscription())
  }

  async function loadPlans(): Promise<void> {
    try {
      account.plans.value = await client.listPlans()
    } catch (failure) {
      account.plans.value = []
      problem.value = getMessage(failure)
    }
  }

  async function refresh(): Promise<void> {
    const before = account.status.value
    try {
      await read()
    } catch (failure) {
      account.status.value = 'unreachable'
      if (getFailureCode(failure) !== 'unavailable') problem.value = getMessage(failure)
      return
    }
    if (before === 'unreachable') problem.value = ''
    if (
      account.status.value === 'ready' &&
      (before !== 'ready' || account.plans.value.length === 0)
    ) {
      await loadPlans()
    }
  }

  function reportProblem(failure: unknown): void {
    problem.value = getMessage(failure)
  }

  function scheduleRefresh(): void {
    clearTimeout(refreshTimer)
    refreshTimer = undefined
    if (isDisposed) return
    const wait =
      account.status.value === 'ready'
        ? refreshMs.ready
        : refreshMs[account.status.value === 'noIdentity' ? 'noIdentity' : 'unreachable']
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

  async function readIfReachable(): Promise<boolean> {
    try {
      await read()
      return true
    } catch {
      return false
    }
  }

  async function change(offer: string): Promise<void> {
    problem.value = ''
    try {
      applyAnswer(account, await client.changePlan(offer))
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
  })

  return { account, server, waiting, problem, load, readIfReachable, change, manage }
}
