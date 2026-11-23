import { defineComponent, h } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import SubscriptionCard from './SubscriptionCard.vue'

export function createSubscriptionView(openExternal: PluginContext['identity']['openExternal']) {
  return defineComponent({
    name: 'SubscriptionView',
    inheritAttrs: false,
    setup: () => () => h(SubscriptionCard, { openExternal }),
  })
}
