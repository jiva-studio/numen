import { describe, expect, it } from 'vitest'
import { createDownloads } from './createDownloads'

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

describe('the downloads of the window', () => {
  it('has a path running from the call until the bytes arrive, and tells both changes', async () => {
    const core = createFetch()
    const downloads = createDownloads(core.fetchFile)
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
    const core = createFetch()
    const downloads = createDownloads(core.fetchFile)

    const first = downloads.fetch('a.mp4')
    const second = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.done()
    await Promise.all([first, second])

    expect(core.calls).toStrictEqual(['a.mp4'])
  })

  it('stops running when the download fails, and rejects with the reason', async () => {
    const core = createFetch()
    const downloads = createDownloads(core.fetchFile)

    const download = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.fail(new Error('offline'))

    await expect(download).rejects.toThrow('offline')
    expect(downloads.isRunning('a.mp4')).toBe(false)
  })

  it('stops telling a handler that was released', async () => {
    const core = createFetch()
    const downloads = createDownloads(core.fetchFile)
    const told: string[] = []
    const stop = downloads.watch((path) => told.push(path))
    stop()

    const download = downloads.fetch('a.mp4')
    core.finish['a.mp4']?.done()
    await download

    expect(told).toStrictEqual([])
  })
})
