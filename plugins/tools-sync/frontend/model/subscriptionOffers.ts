import { computed } from 'vue'
import type { PlanResponse } from '../api/subscription'
import { formatPrice, formatSize, type PlanOffer } from '../lib/subscriptionView'
import type { SubscriptionAccount } from './subscriptionAccount'

const toOffer = (one: PlanResponse): PlanOffer => ({
  id: one.id,
  name: one.name,
  size: formatSize(one.quotaBytes),
  price: formatPrice(one.priceCents, one.currency),
})

/**
 * The plans the person can choose and the actions the account allows: to buy a plan, to change it, to manage it.
 * An active subscription offers every plan to change to, its own plan among them as the current one.
 */
export function useSubscriptionOffers(account: SubscriptionAccount) {
  const canBuy = computed(
    () => account.status.value === 'ready' && account.phase.value !== 'active',
  )
  const buyOffers = computed<readonly PlanOffer[]>(() =>
    canBuy.value ? account.plans.value.map(toOffer) : [],
  )
  const changeOffers = computed<readonly PlanOffer[]>(() =>
    account.status.value === 'ready' && account.phase.value === 'active'
      ? account.plans.value.map(toOffer)
      : [],
  )
  const currentOffer = computed<PlanOffer | undefined>(() =>
    changeOffers.value.find((one) => one.id === account.plan.value),
  )
  const canManage = computed(
    () =>
      account.status.value === 'ready' &&
      account.phase.value !== 'none' &&
      account.phase.value !== '',
  )

  return { canBuy, canManage, buyOffers, changeOffers, currentOffer }
}
