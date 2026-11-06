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

/** The part of `context.commands` the bar runs its commands through. */
interface CommandRunner {
  run(command: string): Promise<void>
}

/** The commands host of the context, typed here until PluginContext declares `commands`. */
function getCommandRunner(context: PluginContext): CommandRunner | undefined {
  return (context as PluginContext & { readonly commands?: CommandRunner }).commands
}

/** The view of the slot: the bar over the entries of its two slots. */
function createStatusBarView(host: SlotHost | undefined, commands: CommandRunner | undefined) {
  return defineComponent({
    name: 'StatusBarView',
    setup: () => {
      const startEntries = useSlotEntries(host, SLOT.statusBarLeft)
      const endEntries = useSlotEntries(host, SLOT.statusBarRight)
      const onRun = (command: string) => {
        if (commands) runInBackground(commands.run(command), reportFailure)
      }
      return () =>
        h(StatusBar, { startEntries: startEntries.value, endEntries: endEntries.value, onRun })
    },
  })
}

export const activate = (context: PluginContext): void => {
  context.registerSlotView({
    id: STATUS_BAR_VIEW,
    component: createStatusBarView(context.slots, getCommandRunner(context)),
  })
}

export default definePlugin({
  activate,
})
