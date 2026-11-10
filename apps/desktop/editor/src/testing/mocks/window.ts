/**
 * What the core answers about the window itself: the vault it shows, its tasks, the focus
 * requests and the quit request. The answers go through the router transport.
 *
 * The background tasks the window lists for its plugins are read from the host the window makes;
 * the module behind the door is wrapped only to hold on to what it made.
 */
import { vi } from 'vitest'
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-loader'
import type { CoreAnswers } from '@numen/core-client/testing'
import { mockVaultList, outside, pendingStream } from '../answers'
import type * as BackgroundTasksModule from '@/features/background-tasks'

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

/** The background tasks the window last made, as its plugins read them. */
const windowTasks = vi.hoisted(() => ({ current: undefined as BackgroundTaskHost | undefined }))

vi.mock('@/features/background-tasks', async (original) => {
  const module = await original<typeof BackgroundTasksModule>()
  return {
    ...module,
    createBackgroundTaskHost: (deps: Parameters<typeof module.createBackgroundTaskHost>[0]) =>
      (windowTasks.current = module.createBackgroundTaskHost(deps)),
  }
})

/** The tasks the window lists for its plugins now; empty before a window is mounted. */
export const readWindowTasks = (): readonly BackgroundTask[] => {
  let tasks: readonly BackgroundTask[] = []
  windowTasks.current?.watch((list) => (tasks = list)).dispose()
  return tasks
}
