import { describe, expect, it, vi } from 'vitest'
import type { ApiError, BackgroundTask, Result } from '@numen/plugin-sdk'
import { createMockBackgroundTaskHost } from '@numen/plugin-sdk/testing'
import type { Stats } from '../api/stats'
import { createIndexStats, REFRESH_MS, type Activity, type Timer } from './stats'

const STATS: Stats = {
  vectors: 10,
  chunks: 10,
  waiting: 0,
  sources: 2,
  kinds: [],
  model: 'bge-m3',
  dimensions: 1024,
  profile: 'balanced',
  threads: 4,
  databaseBytes: 0,
  vectorBytes: 0,
}

const task = (id: string, state: BackgroundTask['state'] = 'running'): BackgroundTask => ({
  id,
  label: 'Indexing',
  about: 'Garden/Roses.md',
  state,
  progress: { completedCount: 3, total: 40, counting: '' },
  error: '',
  allowed: { cancel: true, retry: false },
  actionLabel: '',
})

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}

const setup = (answers: Array<Result<Stats, ApiError>>, isActive = true) => {
  const waits: Array<{ run: () => void; delayMs: number; id: number }> = []
  let nextId = 0
  const timer: Timer = {
    set: (run, delayMs) => {
      waits.push({ run, delayMs, id: nextId })
      return nextId++
    },
    clear: (handle) => {
      const index = waits.findIndex((one) => one.id === handle)
      if (index !== -1) waits.splice(index, 1)
    },
  }
  const state = { isActive }
  const wakes = new Set<() => void>()
  const activity: Activity = {
    isActive: () => state.isActive,
    onActive: (run) => {
      wakes.add(run)
      return () => wakes.delete(run)
    },
  }
  let calls = 0
  const get = vi.fn(async () => answers[Math.min(calls++, answers.length - 1)]!)
  const tasks = createMockBackgroundTaskHost()
  const model = createIndexStats({ api: { get }, timer, activity, tasks })
  const tickAll = () => waits.splice(0).forEach((one) => one.run())
  return { model, get, waits, state, wakes, tickAll, tasks }
}

const ok = (stats: Stats): Result<Stats, ApiError> => ({ ok: true, value: stats })
const failed: Result<Stats, ApiError> = {
  ok: false,
  error: { code: 'failed_precondition', message: 'no model' },
}

describe('the index stats model', () => {
  it('asks once at start and keeps the answer', async () => {
    const { model, get, waits } = setup([ok(STATS)])
    expect(model.phase.value).toBe('loading')

    model.start()
    await flush()

    expect(get).toHaveBeenCalledTimes(1)
    expect(model.stats.value).toEqual(STATS)
    expect(model.phase.value).toBe('ready')
    expect(waits).toEqual([])
  })

  it('does not ask again while the popover is closed and no task starts or ends', async () => {
    const { model, get, waits, wakes, tasks } = setup([ok(STATS)])
    model.start()
    await flush()

    tasks.setTasks([])
    await flush()

    expect(get).toHaveBeenCalledTimes(1)
    expect(waits).toEqual([])
    expect(wakes.size).toBe(0)
  })

  it('asks when a background task starts or ends, and not on its progress', async () => {
    const { model, get, tasks } = setup([ok(STATS), ok({ ...STATS, vectors: 11, chunks: 40 })])
    model.start()
    await flush()

    tasks.setTasks([task('indexing')])
    await flush()
    expect(get).toHaveBeenCalledTimes(2)
    expect(model.stats.value?.vectors).toBe(11)

    tasks.setTasks([
      { ...task('indexing'), progress: { completedCount: 9, total: 40, counting: '' } },
    ])
    await flush()
    expect(get).toHaveBeenCalledTimes(2)

    tasks.setTasks([task('indexing', 'failed')])
    await flush()
    expect(get).toHaveBeenCalledTimes(3)
  })

  it('asks every two seconds while the popover is open and the window is in use', async () => {
    const { model, get, waits, tickAll } = setup([
      ok(STATS),
      ok(STATS),
      ok({ ...STATS, vectors: 11, chunks: 11 }),
    ])
    model.start()
    await flush()

    model.open()
    await flush()
    expect(get).toHaveBeenCalledTimes(2)
    expect(waits.map((one) => one.delayMs)).toEqual([REFRESH_MS])

    tickAll()
    await flush()
    expect(get).toHaveBeenCalledTimes(3)
    expect(model.stats.value?.vectors).toBe(11)
  })

  it('stops the asks every two seconds when the popover closes', async () => {
    const { model, get, waits, wakes } = setup([ok(STATS)])
    model.start()
    await flush()
    model.open()
    await flush()

    model.close()

    expect(waits).toEqual([])
    expect(wakes.size).toBe(0)
    expect(get).toHaveBeenCalledTimes(2)
  })

  it('does not ask with the popover open while the window is out of use, and asks when it returns', async () => {
    const { model, get, state, wakes, tickAll } = setup([ok(STATS)])
    model.start()
    model.open()
    await flush()
    state.isActive = false

    tickAll()
    await flush()
    expect(get).toHaveBeenCalledTimes(1)

    state.isActive = true
    wakes.forEach((wake) => wake())
    await flush()
    expect(get).toHaveBeenCalledTimes(2)
  })

  it('does not start an ask while one is under way', async () => {
    const { model, get, wakes } = setup([ok(STATS)])
    model.start()
    model.open()
    wakes.forEach((wake) => wake())
    await flush()

    expect(get).toHaveBeenCalledTimes(1)
  })

  it('is unavailable where the backend has no index to show, and recovers when it has', async () => {
    const { model, tasks } = setup([failed, ok(STATS)])
    model.start()
    await flush()
    expect(model.phase.value).toBe('unavailable')
    expect(model.stats.value).toBeNull()

    tasks.setTasks([task('indexing')])
    await flush()

    expect(model.phase.value).toBe('ready')
  })

  it('is unavailable where the call itself fails', async () => {
    const model = createIndexStats({
      api: { get: () => Promise.reject(new Error('offline')) },
      timer: { set: () => 0, clear: () => undefined },
      activity: { isActive: () => true, onActive: () => () => undefined },
    })

    model.start()
    await flush()

    expect(model.phase.value).toBe('unavailable')
  })

  it('stops asking and forgets the window and the tasks on stop', async () => {
    const { model, get, waits, wakes, tasks } = setup([ok(STATS)])
    model.start()
    model.open()
    await flush()

    model.stop()
    tasks.setTasks([task('indexing')])
    await flush()

    expect(waits).toEqual([])
    expect(wakes.size).toBe(0)
    expect(get).toHaveBeenCalledTimes(1)
  })

  it('ignores an answer that arrives after stop', async () => {
    const { model } = setup([ok(STATS)])
    model.start()
    model.stop()
    await flush()

    expect(model.stats.value).toBeNull()
  })
})
