/** Which files the plugin's own work is going on, from one shared subscription to its tasks. */
import { describe, expect, it } from 'vitest'
import { createWorkWatcher, type TaskSource } from './work'

function createSource() {
  let handler: ((tasks: readonly { about: string }[]) => void) | null = null
  const counts = { opened: 0, disposed: 0 }
  const source: TaskSource = (onTasks) => {
    counts.opened++
    handler = onTasks
    return {
      dispose: () => {
        counts.disposed++
        handler = null
      },
    }
  }
  const push = (list: readonly { about: string }[]) => handler?.(list)
  return { source, push, counts }
}

describe('a watcher of the plugin work', () => {
  it('reports the first list, then each change of the answer', () => {
    const { source, push } = createSource()
    const changes: boolean[] = []
    createWorkWatcher(source).watch('a.url', (isWorking) => changes.push(isWorking))

    push([])
    push([{ about: 'b.url' }])
    push([{ about: 'a.url' }])
    push([{ about: 'a.url' }])
    push([])

    expect(changes).toStrictEqual([false, true, false])
  })

  it('answers a listener added after the first list at once', () => {
    const { source, push } = createSource()
    const watcher = createWorkWatcher(source)
    watcher.watch('b.url', () => {})
    push([{ about: 'a.url' }])
    const changes: boolean[] = []

    watcher.watch('a.url', (isWorking) => changes.push(isWorking))

    expect(changes).toStrictEqual([true])
  })

  it('opens one subscription for every listener and closes it with the last', () => {
    const { source, counts } = createSource()
    const watcher = createWorkWatcher(source)
    const stopA = watcher.watch('a.url', () => {})
    const stopB = watcher.watch('b.url', () => {})
    stopA()
    stopB()

    expect(counts).toStrictEqual({ opened: 1, disposed: 1 })
  })
})
