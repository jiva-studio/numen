import { describe, expect, it } from 'vitest'
import { createFakeApi, createFakeTimer, settle, SUMMARY } from '../testing/fakeApi'
import { RECONNECT_DELAY_MS, useTaskFeed } from './useTaskFeed'

describe('the task feed', () => {
  it('follows the stream and reads the counts when the first message arrives', async () => {
    const fake = createFakeApi()
    const feed = useTaskFeed(fake.api, () => '2026-11-14', createFakeTimer().timer)

    feed.start()
    fake.push({ revision: 1, isReady: true })
    await settle()

    expect(fake.calls.watchStarts).toBe(1)
    expect(fake.calls.getSummary).toEqual(['2026-11-14'])
    expect(feed.summary.value).toEqual(SUMMARY)
    feed.stop()
  })

  it('reads the counts again on each message and takes the day from the window each time', async () => {
    const fake = createFakeApi()
    let day = '2026-11-14'
    const feed = useTaskFeed(fake.api, () => day, createFakeTimer().timer)
    feed.start()
    fake.push({ revision: 1, isReady: true })
    await settle()

    day = '2026-11-15'
    fake.setSummary({ ...SUMMARY, overdueCount: 9 })
    fake.push({ revision: 2, isReady: true })
    await settle()

    expect(fake.calls.getSummary).toEqual(['2026-11-14', '2026-11-15'])
    expect(feed.summary.value?.overdueCount).toBe(9)
    feed.stop()
  })

  it('reads the counts again at the current day on refresh', async () => {
    const fake = createFakeApi()
    let day = '2026-11-14'
    const feed = useTaskFeed(fake.api, () => day, createFakeTimer().timer)
    feed.start()
    fake.push({ revision: 1, isReady: true })
    await settle()

    day = '2026-11-15'
    await feed.refresh()

    expect(fake.calls.getSummary).toEqual(['2026-11-14', '2026-11-15'])
    feed.stop()
  })

  it('has no counts before the first message', async () => {
    const fake = createFakeApi()
    const feed = useTaskFeed(fake.api, () => '2026-11-14', createFakeTimer().timer)

    feed.start()
    await settle()

    expect(feed.summary.value).toBeNull()
    feed.stop()
  })

  it('ends the stream and reads nothing more after it is stopped', async () => {
    const fake = createFakeApi()
    const feed = useTaskFeed(fake.api, () => '2026-11-14', createFakeTimer().timer)
    feed.start()
    fake.push({ revision: 1, isReady: true })
    await settle()

    feed.stop()
    await settle()
    fake.push({ revision: 2, isReady: true })
    await settle()

    expect(fake.calls.watchEnds).toBe(1)
    expect(fake.calls.getSummary).toEqual(['2026-11-14'])
  })

  it('opens the stream again a second after the backend ended it, and reads the counts then', async () => {
    const fake = createFakeApi()
    const clock = createFakeTimer()
    const feed = useTaskFeed(fake.api, () => '2026-11-14', clock.timer)
    feed.start()
    fake.push({ revision: 1, isReady: true })
    await settle()

    fake.end()
    await settle()
    expect(fake.calls.watchEnds).toBe(1)
    expect(clock.waits.map((one) => one.delayMs)).toEqual([RECONNECT_DELAY_MS])
    expect(RECONNECT_DELAY_MS).toBe(1000)

    clock.fire()
    fake.setSummary({ ...SUMMARY, todayCount: 8 })
    fake.push({ revision: 2, isReady: true })
    await settle()

    expect(fake.calls.watchStarts).toBe(2)
    expect(feed.summary.value?.todayCount).toBe(8)
    feed.stop()
  })

  it('opens no stream again after it is stopped', async () => {
    const fake = createFakeApi()
    const clock = createFakeTimer()
    const feed = useTaskFeed(fake.api, () => '2026-11-14', clock.timer)
    feed.start()
    fake.end()
    await settle()

    feed.stop()
    clock.fire()
    await settle()

    expect(clock.waits).toEqual([])
    expect(fake.calls.watchStarts).toBe(1)
  })

  it('keeps the counts of the last read when an earlier read answers later', async () => {
    const fake = createFakeApi()
    const answers: ((value: typeof SUMMARY) => void)[] = []
    const api = {
      ...fake.api,
      getSummary: () =>
        new Promise<{ ok: true; value: typeof SUMMARY }>((resolve) => {
          answers.push((value) => resolve({ ok: true, value }))
        }),
    }
    const feed = useTaskFeed(api, () => '2026-11-14', createFakeTimer().timer)

    const first = feed.refresh()
    const second = feed.refresh()
    answers[1]?.({ ...SUMMARY, todayCount: 4 })
    answers[0]?.({ ...SUMMARY, todayCount: 1 })
    await Promise.all([first, second])

    expect(feed.summary.value?.todayCount).toBe(4)
  })
})
