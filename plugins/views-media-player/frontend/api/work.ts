/**
 * Which files work is going on, as the window's list of tasks says: one shared stream, open while
 * any tab listens to it.
 */

/** One list of what is being done, by the file each task is about. */
export type TaskStream = (
  signal: AbortSignal,
) => AsyncIterable<readonly { readonly about: string }[]>

/** Tells each listener whether work on its file is going, from one stream of tasks. */
export function createWorkWatcher(openStream: TaskStream) {
  const listeners = new Map<number, { path: string; onChange: (isWorking: boolean) => void }>()
  let nextId = 0
  let working = new Set<string>()
  let abort: AbortController | null = null

  const publish = (paths: Set<string>) => {
    const before = working
    working = paths
    for (const one of listeners.values()) {
      const isWorking = paths.has(one.path)
      if (isWorking || before.has(one.path)) one.onChange(isWorking)
    }
  }

  const open = () => {
    const controller = new AbortController()
    abort = controller
    void (async () => {
      try {
        for await (const tasks of openStream(controller.signal)) {
          publish(new Set(tasks.map((task) => task.about)))
        }
      } catch {
        // A stream that ended is opened again by the next tab that listens.
      } finally {
        if (abort === controller) abort = null
      }
    })()
  }

  return {
    watch(path: string, onChange: (isWorking: boolean) => void): () => void {
      const id = nextId++
      listeners.set(id, { path, onChange })
      if (working.has(path)) onChange(true)
      if (!abort) open()
      return () => {
        listeners.delete(id)
        if (listeners.size === 0 && abort) {
          abort.abort()
          abort = null
          working = new Set()
        }
      }
    },
  }
}
