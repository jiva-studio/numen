import { describe, expect, it } from 'vitest'
import { createFakeApi, settle, SUMMARY } from '../testing/fakeApi'
import { useTaskFeed } from './useTaskFeed'

describe('the task feed', () => {
  it('follows the stream and reads the counts when the first message arrives', async () => {
    const fake = createFakeApi()
    const feed = useTaskFeed(fake.api, () => '2026-11-14')

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
    const feed = useTaskFeed(fake.api, () => day)
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
    const feed = useTaskFeed(fake.api, () => day)
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
    const feed = useTaskFeed(fake.api, () => '2026-11-14')

    feed.start()
    await settle()

    expect(feed.summary.value).toBeNull()
    feed.stop()
  })

  it('ends the stream and reads nothing more after it is stopped', async () => {
    const fake = createFakeApi()
    const feed = useTaskFeed(fake.api, () => '2026-11-14')
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
})
