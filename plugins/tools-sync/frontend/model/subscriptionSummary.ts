import { computed, type Ref } from 'vue'
import {
  formatSize,
  getPlanLine,
  getSubscriptionView,
  getUsagePercent,
  getUsageText,
  getValidity,
  type Validity,
} from '../lib/subscriptionView'
import type { SubscriptionAccount } from './subscriptionAccount'

/** What the card tells of the account: the badge and the text, the plan, the space used and how long it is paid for. */
export function useSubscriptionSummary(account: SubscriptionAccount, server: Ref<string>) {
  const { status, phase, plan, quotaBytes, usedBytes } = account
  const knownPlan = computed(() => account.plans.value.find((one) => one.id === plan.value))
  const view = computed(() =>
    getSubscriptionView({
      status: status.value,
      phase: phase.value,
      dates: account.dates.value,
      server: server.value,
      planName: knownPlan.value?.name ?? '',
    }),
  )
  const planLine = computed(() => {
    if (plan.value === '') return ''
    if (knownPlan.value === undefined) return formatSize(quotaBytes.value)
    const shown = { ...knownPlan.value, quotaBytes: quotaBytes.value }
    return getPlanLine(shown, phase.value !== 'active')
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
  const usedPercent = computed(() => getUsagePercent(usedBytes.value, quotaBytes.value))
  const isFull = computed(() => hasUsage.value && usedBytes.value >= quotaBytes.value)
  const validity = computed<Validity | undefined>(() =>
    status.value === 'ready' && phase.value === 'active'
      ? getValidity(account.dates.value, account.isTrial.value)
      : undefined,
  )

  return { view, planLine, usage, usedPercent, isFull, validity }
}
