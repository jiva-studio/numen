/** Which files work is going on, from one shared subscription to the plugin tasks. */
import { describe, expect, it } from 'vitest'
import { createWorkWatcher, type TaskSource, type WatchedTask } from './work'

/** A task source a test pushes lists through, and how many times it was opened and closed. */
function createSource() {
  let handler: ((tasks: readonly WatchedTask[]) => void) | null = null
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
  const push = (list: readonly WatchedTask[]) => handler?.(list)
  return { source, push, counts }
}

describe('a watcher of work', () => {
  it('reports work on a file when it starts and when it ends', () => {
    const { source, push } = createSource()
    const changes: boolean[] = []
    const watcher = createWorkWatcher(source)
    watcher.watch('a.wav', (isWorking) => changes.push(isWorking))

    push([{ about: 'b.wav' }])
    push([{ about: 'a.wav' }])
    push([{ about: 'a.wav' }, { about: 'b.wav' }])
    push([])

    expect(changes).toStrictEqual([true, true, false])
  })

  it('opens one subscription for every listener, and closes it when the last stops listening', () => {
    const { source, counts } = createSource()
    const watcher = createWorkWatcher(source)
    const stopA = watcher.watch('a.wav', () => undefined)
    const stopB = watcher.watch('b.wav', () => undefined)
    stopA()
    expect(counts.disposed).toBe(0)
    stopB()

    expect(counts.opened).toBe(1)
    expect(counts.disposed).toBe(1)
  })

  it('reports existing work on the file to a new listener', () => {
    const { source, push } = createSource()
    const watcher = createWorkWatcher(source)
    watcher.watch('b.wav', () => undefined)
    push([{ about: 'a.wav' }])
    const changes: boolean[] = []

    watcher.watch('a.wav', (isWorking) => changes.push(isWorking))

    expect(changes).toStrictEqual([true])
  })

  it('ignores tasks that have finished or failed', () => {
    const { source, push } = createSource()
    const changes: boolean[] = []
    const watcher = createWorkWatcher(source)
    watcher.watch('a.wav', (isWorking) => changes.push(isWorking))

    push([{ about: 'a.wav', done: 1, total: 3 }])
    push([{ about: 'a.wav', done: 3, total: 3 }])
    push([{ about: 'a.wav', done: 1, total: 3, error: 'failed' }])

    expect(changes).toStrictEqual([true, false])
  })
})
