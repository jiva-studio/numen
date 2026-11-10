/**
 * The window entity's part of the mocks: its streams from the core, and the background tasks it
 * lists for its plugins.
 */
import { vi } from 'vitest'
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-loader'
import { outside, pendingStream } from '../answers'
import type { VaultCore } from '@/app/vault/core'
import type * as BackgroundTasksModule from '@/app/shell/backgroundTasks'

export const windowCoreMock = {
  watchTasks: pendingStream,
  watchFocus: outside.stream,
  watchQuit: pendingStream,
  reportFlush: () => Promise.resolve(),
} satisfies Partial<VaultCore>

/** The background tasks the window last made, as its plugins read them. */
const windowTasks = vi.hoisted(() => ({ current: undefined as BackgroundTaskHost | undefined }))

vi.mock('@/app/shell/backgroundTasks', async (original) => {
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
