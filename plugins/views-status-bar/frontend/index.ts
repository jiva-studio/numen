/**
 * Frontend entrypoint for the Status bar plugin: the bar at the bottom of the window, drawn in the
 * slot `statusBar`, with the entries of `statusBar.left` and `statusBar.right`.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, SLOT, type PluginContext, type SlotHost } from '@numen/plugin-sdk'
import { reportFailure, runInBackground } from '@numen/ui'
import { useSlotEntries } from './model/useSlotEntries'
import StatusBar from './ui/StatusBar.vue'

/** The id the manifest places the view of the bar under. */
export const STATUS_BAR_VIEW = 'bar'

/** The view of the slot: the bar over the entries of its two slots. */
export function createStatusBarView(host: SlotHost | undefined) {
  return defineComponent({
    name: 'StatusBarView',
    setup: () => {
      const startEntries = useSlotEntries(host, SLOT.statusBarLeft)
      const endEntries = useSlotEntries(host, SLOT.statusBarRight)
      const onRun = (command: string) => {
        if (host) runInBackground(host.runCommand(command), reportFailure)
      }
      return () =>
        h(StatusBar, { startEntries: startEntries.value, endEntries: endEntries.value, onRun })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.registerSlotView({ id: STATUS_BAR_VIEW, component: createStatusBarView(context.slots) })
}

export default definePlugin({
  activate,
})
