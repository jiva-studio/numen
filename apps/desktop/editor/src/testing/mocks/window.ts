/**
 * What the core answers about the window itself: the vault it shows, its tasks, the focus
 * requests and the quit request. The answers go through the router transport.
 */
import type { BackgroundTask, PluginHost } from '@numen/plugin-loader'
import type { CoreAnswers } from '@numen/core-client/testing'
import { mockVaultList, outside, pendingStream } from '../answers'

export const windowAnswers = {
  windowService: {
    getCurrentVault: () => ({ vault: mockVaultList.showing }),
    watchTasks: pendingStream,
    watchQuit: pendingStream,
    reportFlush: () => ({}),
    cancelTask: () => ({}),
  },
  workspace: { watchFocus: outside.stream },
} satisfies CoreAnswers

/** The tasks the window lists for its plugins now, read from the host of its plugins. */
export const readWindowTasks = (host: PluginHost | undefined): readonly BackgroundTask[] => {
  let tasks: readonly BackgroundTask[] = []
  host?.contextHost.window
    .getBackgroundTasks()
    ?.watch((list) => (tasks = list))
    .dispose()
  return tasks
}
