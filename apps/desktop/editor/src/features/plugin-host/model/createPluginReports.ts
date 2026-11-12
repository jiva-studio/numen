import { PLUGIN_FAILURE_WORDS } from '@/entities/plugin'
import { formatError, type Caught } from '@/shared/api/errors'
import type { MessageLog } from '@/shared/model/notices/messages'

/** The name a call to the core gives its errors, whose text formatError replaces by the code's. */
const CONNECT_ERROR = 'ConnectError'

/**
 * Why a plugin did not start. A plain Error comes from the loader or from the plugin bundle and
 * keeps its message; formatError would show it as a lost connection.
 */
const formatLoadError = (error: Caught): string =>
  error instanceof Error && error.name !== CONNECT_ERROR && error.message.trim() !== ''
    ? error.message.trim()
    : formatError(error)

/** What a plugin's problems are written as: the name of the plugin, then the problem, in the log of the plugin. */
export function createPluginReports(
  log: Pick<MessageLog, 'getWriter'>,
  getPluginName: (pluginId: string) => string,
) {
  const write = (pluginId: string, text: string, severity: 'warning' | 'error') =>
    log.getWriter(`plugin:${pluginId}`)(text, severity)
  return {
    writeIssue: (pluginId: string, message: string, severity: 'warning' | 'error') =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, severity),
    reportViewIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'error'),
    reportCommandIssue: (pluginId: string, message: string) =>
      write(pluginId, `${getPluginName(pluginId)}: ${message}`, 'warning'),
    reportError: (pluginId: string, error: Caught) =>
      write(
        pluginId,
        PLUGIN_FAILURE_WORDS.error(getPluginName(pluginId), formatError(error)),
        'error',
      ),
    reportLoadError: (pluginId: string, error: Caught) =>
      write(
        pluginId,
        PLUGIN_FAILURE_WORDS.stopped(
          getPluginName(pluginId),
          `${PLUGIN_FAILURE_WORDS.loadFailed}: ${formatLoadError(error)}`,
        ),
        'error',
      ),
  }
}
