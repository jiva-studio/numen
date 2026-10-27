/**
 * Which files work is going on, as the host's list of plugin tasks says: one shared subscription,
 * open while any tab listens to it.
 */

/** Subscribes to the plugin task lists: each task by the file it is about. */
export type TaskSource = (onTasks: (tasks: readonly { readonly about: string }[]) => void) => {
  dispose(): void
}

/** Tells each listener whether work on its file is going, from one subscription to the tasks. */
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
      subscription ??= watchTasks((tasks) => publish(new Set(tasks.map((task) => task.about))))
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
