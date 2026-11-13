import { runInBackground } from '@numen/ui'
import type { PluginStatus } from '@numen/plugin-loader'
import { formatError } from '@/shared/api/errors'
import type { MessageWriter } from '@/shared/model/notices/messages'

/** A plugin as the settings list it: its id and the file extensions its readers open. */
export interface ReaderPlugin {
  readonly id: string
  readonly readers?: readonly { readonly extensions: readonly string[] }[] | undefined
}

export interface TabPluginStartsDeps {
  readonly status: Pick<PluginStatus, 'startPlugin' | 'startTabPlugins'>
  readonly getPlugins: () => readonly ReaderPlugin[]
  readonly writeMessage: MessageWriter
}

/**
 * The starts of the plugins a tab needs: the plugin of a tab kind, waited for as a file opens in it or
 * started in the background for a placeholder, and the plugins whose readers open an extension.
 */
export function createTabPluginStarts(deps: TabPluginStartsDeps) {
  const { status, writeMessage } = deps
  return {
    startTabKind: (kind: string): Promise<void> => status.startTabPlugins([kind]),
    startKindPlugin: (kind: string): void =>
      runInBackground(status.startTabPlugins([kind]), (error) =>
        writeMessage(`The plugin of the ${kind} tab did not start: ${formatError(error)}`, 'error'),
      ),
    startExtensionReader: async (extension: string): Promise<void> => {
      const plugins = deps
        .getPlugins()
        .filter((one) => (one.readers ?? []).some((reader) => reader.extensions.includes(extension)))
      await Promise.all(plugins.map((one) => status.startPlugin(one.id)))
    },
  }
}
