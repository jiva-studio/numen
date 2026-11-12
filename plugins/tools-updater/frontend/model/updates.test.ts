import { describe, expect, it, vi } from 'vitest'
import type { AppInfo, JsonValue } from '@numen/plugin-sdk'
import type { Status } from '../api/updater'
import {
  createUpdates,
  DEFAULT_INTERVAL_HOURS,
  getIntervalMs,
  RETRY_DELAYS_MS,
  SETTLE_DELAY_MS,
  type Timer,
} from './updates'

const APP: AppInfo = { name: 'numen', version: '2026.9.0', build: 900, platform: 'windows' }
const HOUR = 60 * 60 * 1000

const AVAILABLE: Status = {
  state: 'available',
  update: {
    version: '2026.10.1',
    build: 950,
    channel: 'stable',
    releasedAt: '2026-10-15T12:00:00Z',
    notesUrl: 'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
    downloadUrl: 'https://dl.numen.md/numen-windows-setup.exe',
  },
}

const createFakeTimer = () => {
  const waits = new Map<number, { run: () => void; delayMs: number }>()
  let next = 1
  const timer: Timer = {
    set: (run, delayMs) => {
      const handle = next++
      waits.set(handle, { run, delayMs })
      return handle
    },
    clear: (handle) => {
      waits.delete(handle)
    },
  }
  return {
    timer,
    delays: () => [...waits.values()].map((one) => one.delayMs),
    runNext: async () => {
      const [handle, wait] = [...waits.entries()][0] ?? []
      if (handle === undefined || !wait) throw new Error('no wait is pending')
      waits.delete(handle)
      wait.run()
      await flush()
    },
  }
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await Promise.resolve()
}

const create = (answers: Array<Status | 'fail' | 'reject'>, hours: JsonValue | undefined = 4) => {
  const clock = createFakeTimer()
  const queue = [...answers]
  const check = vi.fn(async () => {
    const answer = queue.shift() ?? queue[queue.length - 1]
    if (answer === 'reject') throw new Error('no connection')
    if (answer === 'fail' || answer === undefined) {
      return { ok: false as const, error: { code: 'unavailable', message: 'down' } }
    }
    return { ok: true as const, value: answer }
  })
  const openExternal = vi.fn(async () => ({ ok: true as const, value: null }))
  const updates = createUpdates({
    api: { check },
    app: APP,
    openExternal,
    getIntervalHours: () => hours,
    timer: clock.timer,
  })
  return { updates, clock, check, openExternal }
}

describe('the interval between checks', () => {
  it('is the hours of the setting', () => {
    expect(getIntervalMs(2)).toBe(2 * HOUR)
    expect(getIntervalMs('6')).toBe(6 * HOUR)
  })

  it('is the default where the setting holds no usable number', () => {
    for (const bad of [undefined, 'abc', 0, -3, 0.5, Number.NaN]) {
      expect(getIntervalMs(bad)).toBe(DEFAULT_INTERVAL_HOURS * HOUR)
    }
  })
})

describe('starting the checks', () => {
  it('checks the running build at once and announces what it finds', async () => {
    const { updates, check } = create([AVAILABLE])

    updates.schedule.start()
    await flush()

    expect(check).toHaveBeenCalledWith(APP)
    expect(updates.announced.value?.version).toBe('2026.10.1')
  })

  it('waits the interval of the setting before the next check', async () => {
    const { updates, clock, check } = create([AVAILABLE], 2)
    updates.schedule.start()
    await flush()

    expect(clock.delays()).toEqual([2 * HOUR])
    await clock.runNext()

    expect(check).toHaveBeenCalledTimes(2)
  })

  it('announces nothing for a build that is current', async () => {
    const { updates } = create([{ state: 'current' }])

    updates.schedule.start()
    await flush()

    expect(updates.announced.value).toBeNull()
  })
})

describe('a check that reaches no answer', () => {
  it('shows nothing and is repeated after growing waits', async () => {
    const { updates, clock, check } = create(['fail'])
    updates.schedule.start()
    await flush()

    expect(updates.announced.value).toBeNull()
    expect(clock.delays()).toEqual([RETRY_DELAYS_MS[0]])
    await clock.runNext()
    expect(clock.delays()).toEqual([RETRY_DELAYS_MS[1]])
    await clock.runNext()
    await clock.runNext()
    await clock.runNext()

    expect(clock.delays()).toEqual([RETRY_DELAYS_MS[2]])
    expect(check).toHaveBeenCalledTimes(5)
  })

  it('is treated as unreachable when the backend says so', async () => {
    const { updates, clock } = create([{ state: 'unreachable' }, AVAILABLE])
    updates.schedule.start()
    await flush()

    expect(updates.announced.value).toBeNull()
    await clock.runNext()

    expect(updates.announced.value?.version).toBe('2026.10.1')
  })

  it('is repeated when the call itself rejects', async () => {
    const { updates, clock } = create(['reject'])

    updates.schedule.start()
    await flush()

    expect(updates.announced.value).toBeNull()
    expect(clock.delays()).toEqual([RETRY_DELAYS_MS[0]])
  })

  it('is never repeated later than the interval', async () => {
    const { updates, clock } = create(['fail'], 1)
    updates.schedule.start()
    await flush()
    for (let i = 0; i < 4; i++) await clock.runNext()

    expect(clock.delays()[0]).toBeLessThanOrEqual(HOUR)
  })
})

describe('dismissing the announcement', () => {
  it('hides the update found, and shows a newer one', async () => {
    const newer: Status = {
      state: 'available',
      update: { ...(AVAILABLE.update as NonNullable<Status['update']>), version: '2026.10.2' },
    }
    const { updates, clock } = create([AVAILABLE, AVAILABLE, newer])
    updates.schedule.start()
    await flush()

    updates.announcement.dismiss()
    expect(updates.announced.value).toBeNull()
    await clock.runNext()
    expect(updates.announced.value).toBeNull()
    await clock.runNext()

    expect(updates.announced.value?.version).toBe('2026.10.2')
  })
})

describe('opening the pages of the update', () => {
  it('opens the download and the release notes in the system browser', async () => {
    const { updates, openExternal } = create([AVAILABLE])
    updates.schedule.start()
    await flush()

    await updates.announcement.openDownload()
    await updates.announcement.openNotes()

    expect(openExternal).toHaveBeenNthCalledWith(1, 'https://dl.numen.md/numen-windows-setup.exe')
    expect(openExternal).toHaveBeenNthCalledWith(
      2,
      'https://github.com/jiva-studio/numen/releases/tag/v2026.10.1',
    )
  })

  it('opens nothing where nothing is announced', async () => {
    const { updates, openExternal } = create([{ state: 'current' }])

    expect(await updates.announcement.openDownload()).toEqual({ ok: true, value: null })
    expect(openExternal).not.toHaveBeenCalled()
  })
})

describe('changing the settings', () => {
  it('counts the interval again from the change', async () => {
    const { updates, clock } = create([AVAILABLE], 3)
    updates.schedule.start()
    await flush()

    updates.schedule.reschedule()

    expect(clock.delays()).toEqual([3 * HOUR])
  })

  it('checks soon after the feed changed, from an unchecked state', async () => {
    const { updates, clock, check } = create([AVAILABLE, { state: 'current' }])
    updates.schedule.start()
    await flush()

    updates.schedule.checkSoon()

    expect(updates.status.value.state).toBe('unchecked')
    expect(clock.delays()).toEqual([SETTLE_DELAY_MS])
    await clock.runNext()
    expect(check).toHaveBeenCalledTimes(2)
    expect(updates.announced.value).toBeNull()
  })
})

describe('stopping the checks', () => {
  it('ends the wait and ignores an answer that arrives later', async () => {
    const { updates, clock } = create([AVAILABLE])
    updates.schedule.start()
    updates.schedule.stop()
    await flush()

    expect(clock.delays()).toEqual([])
    expect(updates.announced.value).toBeNull()
  })
})
