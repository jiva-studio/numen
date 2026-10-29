/**
 * Tracks which files have work in progress according to the plugin's tasks,
 * through one shared subscription that stays open while any tab listens.
 */

/** Subscribes to the plugin's task lists: each task by the file it is about. */
export type TaskSource = (onTasks: (tasks: readonly { readonly about: string }[]) => void) => {
  dispose(): void
}

/**
 * Notifies each listener whether work on its file is in progress, from one
 * subscription to the task lists: on the first list received, and on every
 * later list that changes the result.
 */
export function createWorkWatcher(watchTasks: TaskSource) {
  const listeners = new Map<
    number,
    { path: string; isHeard: boolean; onChange: (isWorking: boolean) => void }
  >()
  let nextId = 0
  let working = new Set<string>()
  let hasList = false
  let subscription: { dispose(): void } | null = null

  const publish = (paths: Set<string>) => {
    const before = working
    working = paths
    hasList = true
    for (const one of listeners.values()) {
      const isWorking = paths.has(one.path)
      if (!one.isHeard || isWorking !== before.has(one.path)) {
        one.isHeard = true
        one.onChange(isWorking)
      }
    }
  }

  return {
    watch(path: string, onChange: (isWorking: boolean) => void): () => void {
      const id = nextId++
      listeners.set(id, { path, isHeard: hasList, onChange })
      if (hasList) onChange(working.has(path))
      subscription ??= watchTasks((tasks) => publish(new Set(tasks.map((task) => task.about))))
      return () => {
        listeners.delete(id)
        if (listeners.size === 0 && subscription) {
          subscription.dispose()
          subscription = null
          working = new Set()
          hasList = false
        }
      }
    },
    refresh(paths: readonly string[]): void {
      for (const one of listeners.values()) {
        if ((paths.length === 0 || paths.includes(one.path)) && !working.has(one.path)) {
          one.onChange(false)
        }
      }
    },
  }
}
