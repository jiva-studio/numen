/**
 * Which files have work in progress, according to the host's list of plugin tasks: one shared
 * subscription, open while any tab is subscribed to it.
 */

/** One task reported by the host: the file it is on, its progress, and any error. */
export interface WatchedTask {
  readonly about: string
  readonly done?: number | undefined
  readonly total?: number | undefined
  readonly error?: string | undefined
}

/** Subscribes to the plugin task lists: each task by the file it is about. */
export type TaskSource = (onTasks: (tasks: readonly WatchedTask[]) => void) => {
  dispose(): void
}

const isActiveTask = (task: WatchedTask): boolean =>
  !task.error && ((task.total ?? 0) === 0 || (task.done ?? 0) < (task.total ?? 0))

/** Notifies each listener whether work on its file is in progress, from one subscription to the tasks. */
export function createWorkWatcher(watchTasks: TaskSource) {
  const listeners = new Map<number, { path: string; onChange: (isWorking: boolean) => void }>()
  let nextId = 0
  let working = new Set<string>()
  let subscription: { dispose(): void } | null = null

  const publish = (paths: Set<string>) => {
    const before = working
    working = paths
    for (const one of listeners.values()) {
      const isWorking = paths.has(one.path)
      if (isWorking || before.has(one.path)) one.onChange(isWorking)
    }
  }

  return {
    watch(path: string, onChange: (isWorking: boolean) => void): () => void {
      const id = nextId++
      listeners.set(id, { path, onChange })
      if (working.has(path)) onChange(true)
      subscription ??= watchTasks((tasks) =>
        publish(new Set(tasks.filter(isActiveTask).map((task) => task.about))),
      )
      return () => {
        listeners.delete(id)
        if (listeners.size === 0 && subscription) {
          subscription.dispose()
          subscription = null
          working = new Set()
        }
      }
    },
  }
}
