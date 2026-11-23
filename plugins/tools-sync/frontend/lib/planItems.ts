import { defineComponent, h, markRaw } from 'vue'
import type { MenuItem } from '@numen/ui'
import { WORDS as words } from '../words'
import { getOfferDetail, type PlanOffer } from './subscriptionView'

const getCurrentName = (name: string) =>
  markRaw(
    defineComponent({
      name: 'CurrentPlanName',
      setup: () => () => [
        h('span', name),
        h('span', { class: 'subscription__current-tag' }, words.currentPlan),
      ],
    }),
  )

/** The items of the menu of plans: every plan with its size and price, the current one tagged and not to be chosen. */
export function getPlanItems(offers: readonly PlanOffer[], currentId: string): readonly MenuItem[] {
  return offers.map((offer) => ({
    id: offer.id,
    text: offer.name,
    detail: getOfferDetail(offer),
    ...(offer.id === currentId ? { isDisabled: true, component: getCurrentName(offer.name) } : {}),
  }))
}
