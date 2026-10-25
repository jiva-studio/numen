/** Which files work is going on, from one shared stream of the window's tasks. */
import { describe, expect, it } from 'vitest'
import { createWorkWatcher, type TaskStream } from './work'

/** A stream of task lists a test pushes, and how many times it was opened and closed. */
function createStream() {
  const lists: (readonly { id: string; about: string }[])[] = []
  let wake: (() => void) | null = null
  const counts = { opened: 0, aborted: 0 }
  const stream: TaskStream = async function* (signal) {
    counts.opened++
    signal.addEventListener('abort', () => {
      counts.aborted++
      wake?.()
    })
    while (!signal.aborted) {
      const next = lists.shift()
      if (next) {
        yield next
        continue
      }
      await new Promise<void>((done) => (wake = done))
    }
  }
  const push = (list: readonly { id: string; about: string }[]) => {
    lists.push(list)
    wake?.()
  }
  return { stream, push, counts }
}

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('a watcher of work', () => {
  it('reports work on a file when it starts and when it ends', async () => {
    const { stream, push } = createStream()
    const changes: boolean[] = []
    const watcher = createWorkWatcher(stream)
    watcher.watch('a.wav', (isWorking) => changes.push(isWorking))

    push([{ id: 'plugin:p:t', about: 'b.wav' }])
    await flush()
    push([{ id: 'plugin:p:t', about: 'a.wav' }])
    await flush()
    push([
      { id: 'plugin:p:t', about: 'a.wav' },
      { id: 'plugin:p:t', about: 'b.wav' },
    ])
    await flush()
    push([])
    await flush()

    expect(changes).toStrictEqual([true, true, false])
  })

  it('ignores a task the core lists about the file itself', async () => {
    const { stream, push } = createStream()
    const changes: boolean[] = []
    createWorkWatcher(stream).watch('a.wav', (isWorking) => changes.push(isWorking))

    push([{ id: 'making the vectors', about: 'a.wav' }])
    await flush()

    expect(changes).toStrictEqual([])
  })

  it('opens one stream for every listener, and closes it when the last stops listening', async () => {
    const { stream, counts } = createStream()
    const watcher = createWorkWatcher(stream)
    const stopA = watcher.watch('a.wav', () => {})
    const stopB = watcher.watch('b.wav', () => {})
    await flush()
    stopA()
    expect(counts.aborted).toBe(0)
    stopB()
    await flush()

    expect(counts.opened).toBe(1)
    expect(counts.aborted).toBe(1)
  })
})
