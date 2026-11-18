/**
 * Frontend entrypoint for the Logs plugin: registers the `logs` tab kind and the `logs.open`
 * command.
 */
import { onScopeDispose } from 'vue'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import { createLogsView, type LogsView } from './model/logs'
import LogsTab from './ui/LogsTab.vue'
import { WORDS as words } from './words'
import './styles.css'

/** The tab kind id declared in `manifest.json`. */
export const LOGS_KIND = 'logs'

/** The command id declared in `manifest.json`. */
export const OPEN_LOGS_COMMAND = 'logs.open'

export function activate(plugin: PluginContext, host: HostServices): void {
  plugin.register.tabKind<LogsView>({
    kind: LOGS_KIND,
    title: words.title,
    component: LogsTab,
    isTool: true,
    identity: () => LOGS_KIND,
    open: () => {
      const controller = createLogsView(host.work.logs)
      controller.start()
      onScopeDispose(() => controller.stop())
      return controller.view
    },
  })

  plugin.register.command({
    id: OPEN_LOGS_COMMAND,
    handler: async () => {
      await host.tabs.open?.(LOGS_KIND)
    },
  })
}

export default definePlugin({
  activate,
})
