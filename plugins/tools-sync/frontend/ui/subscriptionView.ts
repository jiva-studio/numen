import { defineComponent, h } from 'vue'
import type { HostServices, PluginContext } from '@numen/plugin-sdk'
import type { AccountClient } from '../api/account'
import { createSaveFile } from '../api/saveFile'
import SubscriptionCard from './SubscriptionCard.vue'

export function createSubscriptionView(
  plugin: Pick<PluginContext, 'settings'> & {
    identity: Pick<PluginContext['identity'], 'openExternal'>
  },
  host: Pick<HostServices, 'ui'>,
  accountClient: AccountClient,
  openSettings?: (() => void) | undefined,
) {
  const saveFile = createSaveFile(host.ui.dialogs)
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
