import { describe, expect, it, vi } from 'vitest'
import type { DownloadProgress } from '@numen/core-client'
import { asValue, type Result } from '@numen/result'
import { createDownloads, PROGRESS_INTERVAL_MS } from './createDownloads'

type Answer = Result<DownloadProgress>
type Reader = (path: string) => Promise<Answer>

const createFetch = () => {
  const finish: Record<string, { done: () => void; fail: (error: Error) => void }> = {}
  const calls: string[] = []
  return {
    calls,
    finish,
    fetchFile: (path: string) => {
      calls.push(path)
      return new Promise<void>((done, fail) => {
        finish[path] = { done, fail }
      })
    },
  }
}

const createTimers = () => {
  const tasks = new Set<() => void>()
  return {
    tasks,
    ports: {
      every: (task: () => void, ms: number) => {
        expect(ms).toBe(PROGRESS_INTERVAL_MS)
        tasks.add(task)
        return () => tasks.delete(task)
      },
    },
    tick: () => tasks.forEach((task) => task()),
  }
}

const readUnknown: Reader = () =>
  Promise.resolve(asValue({ isKnown: false, bytesDone: 0, bytesTotal: 0 }))

const create = (readProgress: Reader = readUnknown) => {
  const core = createFetch()
  const timers = createTimers()
  return {
    core,
    timers,
    downloads: createDownloads(core.fetchFile, readProgress, timers.ports),
  }
}

describe('the downloads of the window', () => {
  it('has a path running from the call until the bytes arrive, and tells both changes', async () => {
    const { core, downloads } = create()
    const told: string[] = []
    downloads.watch((path) => told.push(`${path} ${downloads.isRunning(path)}`))

    const download = downloads.fetch('Lecture.mp4')
    expect(downloads.isRunning('Lecture.mp4')).toBe(true)
    core.finish['Lecture.mp4']?.done()
    await download

    expect(downloads.isRunning('Lecture.mp4')).toBe(false)
    expect(told).toStrictEqual(['Lecture.mp4 true', 'Lecture.mp4 false'])
  })

  it('joins a second call for a path already running', async () => {
    const { core, downloads } = create()

    const first = downloads.fetch('a.mp4')
    const second = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.done()
    await Promise.all([first, second])

    expect(core.calls).toStrictEqual(['a.mp4'])
  })

  it('stops running when the download fails, and rejects with the reason', async () => {
    const { core, downloads } = create()

    const download = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.fail(new Error('offline'))

    await expect(download).rejects.toThrow('offline')
    expect(downloads.isRunning('a.mp4')).toBe(false)
  })

  it('stops telling a handler that was released', async () => {
    const { core, downloads } = create()
    const told: string[] = []
    const stop = downloads.watch((path) => told.push(path))
    stop()

    const download = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.done()
    await download

    expect(told).toStrictEqual([])
  })
})

describe('the progress of a download', () => {
  it('is read on each tick while the download runs, and told when it moves on', async () => {
    const answers: DownloadProgress[] = [
      { isKnown: false, bytesDone: 0, bytesTotal: 0 },
      { isKnown: true, bytesDone: 12, bytesTotal: 41 },
      { isKnown: true, bytesDone: 12, bytesTotal: 41 },
    ]
    const readProgress = vi.fn<Reader>(() => Promise.resolve(asValue(answers.shift()!)))
    const { core, downloads, timers } = create(readProgress)
    const told: string[] = []
    downloads.watch((path) => told.push(path))

    const download = downloads.fetch('a.mp4')
    expect(downloads.getProgress('a.mp4')).toBeUndefined()

    timers.tick()
    await vi.waitFor(() => expect(readProgress).toHaveBeenCalledTimes(1))
    expect(downloads.getProgress('a.mp4')).toBeUndefined()

    timers.tick()
    await vi.waitFor(() =>
      expect(downloads.getProgress('a.mp4')).toEqual({ bytesDone: 12, bytesTotal: 41 }),
    )
    expect(readProgress).toHaveBeenCalledWith('a.mp4')

    timers.tick()
    await vi.waitFor(() => expect(readProgress).toHaveBeenCalledTimes(3))
    expect(told).toStrictEqual(['a.mp4', 'a.mp4'])

    core.finish['a.mp4']?.done()
    await download
  })

  it('stops being read, and is forgotten, when the download ends', async () => {
    const { core, downloads, timers } = create(() =>
      Promise.resolve(asValue({ isKnown: true, bytesDone: 1, bytesTotal: 2 })),
    )

    const download = downloads.fetch('a.mp4')
    timers.tick()
    await vi.waitFor(() => expect(downloads.getProgress('a.mp4')).toBeDefined())
    expect(timers.tasks.size).toBe(1)

    core.finish['a.mp4']?.done()
    await download

    expect(timers.tasks.size).toBe(0)
    expect(downloads.getProgress('a.mp4')).toBeUndefined()
  })

  it('does not start a second read while one is waiting', () => {
    const readProgress = vi.fn<Reader>(() => new Promise<Answer>(() => undefined))
    const { downloads, timers } = create(readProgress)

    downloads.fetch('a.mp4').catch(() => undefined)
    timers.tick()
    timers.tick()

    expect(readProgress).toHaveBeenCalledTimes(1)
  })
})
