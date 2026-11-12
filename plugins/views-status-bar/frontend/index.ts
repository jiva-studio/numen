/**
 * Frontend entrypoint for the Status bar plugin: the bar at the bottom of the window, drawn in the
 * slot `statusBar`, with the entries of `statusBar.left` and `statusBar.right`.
 */
import { defineComponent, h } from 'vue'
import { definePlugin, SLOT, type PluginContext, type Caught } from '@numen/plugin-sdk'
import { runInBackground } from '@numen/ui'
import { useSlotEntries } from './model/useSlotEntries'
import StatusBar from './ui/StatusBar.vue'

/** The id the manifest places the view of the bar under. */
export const STATUS_BAR_VIEW = 'bar'

function createStatusBarView(context: PluginContext) {
  const { slots: host, commands } = context.ui
  const reportError = (error: Caught): void =>
    context.ui.notices.show(error instanceof Error ? error.message : String(error), 'error')
  return defineComponent({
    name: 'StatusBarView',
    setup: () => {
      const startEntries = useSlotEntries(host, SLOT.statusBarLeft)
      const endEntries = useSlotEntries(host, SLOT.statusBarRight)
      const onRun = (
        command: string,
        itemId?: string,
        values?: Readonly<Record<string, string>>,
      ) => {
        if (commands) {
          runInBackground(commands.run(command, undefined, itemId, values), reportError)
        }
      }
      const listItems = (command: string) => commands?.listItems(command) ?? Promise.resolve([])
      return () =>
        h(StatusBar, {
          startEntries: startEntries.value,
          endEntries: endEntries.value,
          listItems,
          reportError,
          onRun,
        })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.register.slotView({
    id: STATUS_BAR_VIEW,
    component: createStatusBarView(context),
  })
}

export default definePlugin({
  activate,
})
