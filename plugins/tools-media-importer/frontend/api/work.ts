/**
 * Which files work is going on, as the window's list of tasks says: one shared stream, open while
 * any tab listens to it.
 */

/** One list of what is being done: each task by its id and the file it is about. */
export type TaskStream = (
  signal: AbortSignal,
) => AsyncIterable<readonly { readonly id: string; readonly about: string }[]>

/** The core lists a plugin's task under an id starting with this; the index's own tasks do not. */
const PLUGIN_TASK_PREFIX = 'plugin:'

/**
 * Tells each listener whether work on its file is going, from one stream of tasks: on the first
 * list it hears, and on every list after that changes the answer.
 */
export function createWorkWatcher(openStream: TaskStream) {
  const listeners = new Map<
    number,
    { path: string; isHeard: boolean; onChange: (isWorking: boolean) => void }
  >()
  let nextId = 0
  let working = new Set<string>()
  let abort: AbortController | null = null

  const publish = (paths: Set<string>) => {
    const before = working
    working = paths
    for (const one of listeners.values()) {
      const isWorking = paths.has(one.path)
      if (!one.isHeard || isWorking !== before.has(one.path)) {
        one.isHeard = true
        one.onChange(isWorking)
      }
    }
  }

  const open = () => {
    const controller = new AbortController()
    abort = controller
    void (async () => {
      try {
        for await (const tasks of openStream(controller.signal)) {
          publish(
            new Set(
              tasks
                .filter((task) => task.id.startsWith(PLUGIN_TASK_PREFIX))
                .map((task) => task.about),
            ),
          )
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
      listeners.set(id, { path, isHeard: false, onChange })
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
