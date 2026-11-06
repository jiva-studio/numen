/**
 * Frontend entrypoint for the Status bar plugin: the bar at the bottom of the window, drawn in the
 * slot `statusBar`, with the entries of `statusBar.left` and `statusBar.right`.
 */
import { defineComponent, h } from 'vue'
import {
  definePlugin,
  SLOT,
  type CommandsHost,
  type PluginContext,
  type SlotHost,
} from '@numen/plugin-sdk'
import { reportFailure, runInBackground } from '@numen/ui'
import { useSlotEntries } from './model/useSlotEntries'
import StatusBar from './ui/StatusBar.vue'

/** The id the manifest places the view of the bar under. */
export const STATUS_BAR_VIEW = 'bar'

/** The view of the slot: the bar over the entries of its two slots. */
function createStatusBarView(host: SlotHost | undefined, commands: CommandsHost | undefined) {
  return defineComponent({
    name: 'StatusBarView',
    setup: () => {
      const startEntries = useSlotEntries(host, SLOT.statusBarLeft)
      const endEntries = useSlotEntries(host, SLOT.statusBarRight)
      const onRun = (command: string, argument?: string) => {
        if (commands) runInBackground(commands.run(command, undefined, argument), reportFailure)
      }
      const listItems = (command: string) => commands?.listItems(command) ?? Promise.resolve([])
      return () =>
        h(StatusBar, {
          startEntries: startEntries.value,
          endEntries: endEntries.value,
          listItems,
          onRun,
        })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.registerSlotView({
    id: STATUS_BAR_VIEW,
    component: createStatusBarView(context.slots, context.commands),
  })
}

export default definePlugin({
  activate,
})
