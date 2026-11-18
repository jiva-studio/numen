import { defineComponent, h } from 'vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { createSupporterState, SUPPORTER_LICENSE_ID, type SupporterState } from './model/supporter'
import SupporterBadge from './ui/SupporterBadge.vue'
import './styles.css'

export const SUPPORTER_VIEW = SUPPORTER_LICENSE_ID

export function createSupporterView(state: SupporterState) {
  return defineComponent({
    name: 'SupporterStatusView',
    setup: () => () =>
      h(SupporterBadge, {
        isActive: state.isActive.value,
      }),
  })
}

export const activate = (plugin: PluginContext): void => {
  const state = createSupporterState(plugin.licenses)
  plugin.register.slotView({ id: SUPPORTER_VIEW, component: createSupporterView(state) })
  plugin.identity.subscriptions.push({ dispose: () => state.stop() })
  state.start()
}

export default definePlugin({
  activate,
})
