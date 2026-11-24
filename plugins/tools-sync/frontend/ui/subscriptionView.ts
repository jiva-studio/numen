import { defineComponent, h } from 'vue'
import type { PluginContext } from '@numen/plugin-sdk'
import { accountClient } from '../api/wire'
import { saveFile } from '../api/saveFile'
import SubscriptionCard from './SubscriptionCard.vue'

export function createSubscriptionView(
  plugin: Pick<PluginContext, 'settings'> & {
    identity: Pick<PluginContext['identity'], 'openExternal'>
  },
  openSettings?: (() => void) | undefined,
) {
  return defineComponent({
    name: 'SubscriptionView',
    inheritAttrs: false,
    setup: () => () =>
      h(SubscriptionCard, {
        openExternal: plugin.identity.openExternal,
        settings: plugin.settings,
        accountClient,
        saveFile,
        openSettings,
      }),
  })
}
