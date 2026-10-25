/** Which files work is going on, from one shared stream of the window's tasks. */
import { describe, expect, it } from 'vitest'
import { createWorkWatcher, type TaskStream } from './work'

/** A stream of task lists a test pushes, and how many times it was opened and closed. */
function createStream() {
  const lists: (readonly { about: string }[])[] = []
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
  const push = (list: readonly { about: string }[]) => {
    lists.push(list)
    wake?.()
  }
  return { stream, push, counts }
}

const flush = () => new Promise((done) => setTimeout(done, 0))

describe('a watcher of work', () => {
  it('tells a file it is being worked on, and once more when the work ends', async () => {
    const { stream, push } = createStream()
    const told: boolean[] = []
    const watcher = createWorkWatcher(stream)
    watcher.watch('a.wav', (isWorking) => told.push(isWorking))

    push([{ about: 'b.wav' }])
    await flush()
    push([{ about: 'a.wav' }])
    await flush()
    push([{ about: 'a.wav' }, { about: 'b.wav' }])
    await flush()
    push([])
    await flush()

    expect(told).toStrictEqual([true, true, false])
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
