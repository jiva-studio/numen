import { watch } from 'vue'
import { runInBackground } from '@numen/ui'
import type { PluginLoader } from '@numen/plugin-loader'
import { getFrontendPluginIds } from '@/entities/plugin'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'

/** The plugins the settings name, as the window reads them. */
type SettingPlugins = () => readonly { readonly id: string; readonly hasFrontend: boolean }[]

export interface PluginStartDeps {
  readonly pluginLoader: PluginLoader
  readonly getPlugins: SettingPlugins
  readonly waitForDeclaredCommands: () => Promise<void>
  readonly writeMessage: MessageWriter
}

/** Loads the frontends of the plugins the settings name, and again as the settings change them. */
export function createPluginStart(deps: PluginStartDeps) {
  const { pluginLoader } = deps
  watch(deps.getPlugins, (plugins) => {
    runInBackground(pluginLoader.trackPlugins(getFrontendPluginIds(plugins)), (error) =>
      deps.writeMessage(`Plugins were not tracked: ${formatError(error)}`, 'error'),
    )
  })
  return {
    /** The plugins the settings name are active before the layout opens the tabs of their kinds. */
    start: async (): Promise<void> => {
      await pluginLoader.trackPlugins(getFrontendPluginIds(deps.getPlugins()))
      await pluginLoader.start()
    },
    // The title bar and the shortcuts read the commands the plugins declare, so the window waits for
    // their first list as it waits for the plugins of the slots of the window.
    waitForInterface: async (): Promise<void> => {
      await Promise.all([pluginLoader.status.waitForInterface(), deps.waitForDeclaredCommands()])
    },
    close: async (): Promise<void> => {
      await pluginLoader.close()
    },
  }
}
