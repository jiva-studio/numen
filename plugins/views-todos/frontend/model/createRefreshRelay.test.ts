import { describe, expect, it } from 'vitest'
import { createFakeApi, createFakeTimer, settle } from '../testing/fakeApi'
import { createRefreshRelay, REFRESH_DELAY_MS } from './createRefreshRelay'

const setup = () => {
  const fake = createFakeApi()
  const clock = createFakeTimer()
  let refreshed = 0
  const relay = createRefreshRelay(fake.api, clock.timer, () => {
    refreshed += 1
  })
  return { fake, clock, relay, getRefreshed: () => refreshed }
}

describe('the refresh relay', () => {
  it('sets one wait of 400 milliseconds however many events come', () => {
    const { clock, relay } = setup()

    relay.push({ paths: ['a.md'] })
    relay.push({ paths: ['b.md'] })

    expect(REFRESH_DELAY_MS).toBe(400)
    expect(clock.waits.map((one) => one.delayMs)).toEqual([400])
  })

  it('sends the paths and renames it collected once, when the wait ends', () => {
    const { fake, clock, relay } = setup()

    relay.push({ paths: ['a.md'] })
    relay.push({ paths: ['b.md', 'a.md'], renamed: [{ from: 'old.md', to: 'new.md' }] })
    expect(fake.calls.refreshIndex).toEqual([])
    clock.fire()

    expect(fake.calls.refreshIndex).toEqual([
      { paths: ['a.md', 'b.md'], renamed: [{ from: 'old.md', to: 'new.md' }] },
    ])
  })

  it('starts a new collection after it sent one', () => {
    const { fake, clock, relay } = setup()
    relay.push({ paths: ['a.md'] })
    clock.fire()

    relay.push({ paths: ['b.md'] })
    clock.fire()

    expect(fake.calls.refreshIndex).toEqual([
      { paths: ['a.md'], renamed: [] },
      { paths: ['b.md'], renamed: [] },
    ])
  })

  it('asks for a full check at once when an event names no path and no rename', () => {
    const { fake, clock, relay } = setup()

    relay.push({})
    relay.push({ paths: [], renamed: [] })

    expect(fake.calls.refreshIndex).toEqual([
      { paths: [], renamed: [] },
      { paths: [], renamed: [] },
    ])
    expect(clock.waits).toEqual([])
  })

  it('sends no second call for paths it collected before a full check', () => {
    const { fake, clock, relay } = setup()
    relay.push({ paths: ['a.md'] })

    relay.push({})
    clock.fire()

    expect(fake.calls.refreshIndex).toEqual([{ paths: [], renamed: [] }])
  })

  it('sends nothing after it is disposed', () => {
    const { fake, clock, relay } = setup()
    relay.push({ paths: ['a.md'] })

    relay.dispose()
    clock.fire()

    expect(clock.waits).toEqual([])
    expect(fake.calls.refreshIndex).toEqual([])
  })

  it('tells once the backend has checked the paths it sent, and not before', async () => {
    const { clock, relay, getRefreshed } = setup()
    relay.push({ paths: ['a.md'] })
    await settle()
    expect(getRefreshed()).toBe(0)

    clock.fire()
    await settle()

    expect(getRefreshed()).toBe(1)
  })

  it('tells once the backend has made a full check', async () => {
    const { relay, getRefreshed } = setup()

    relay.push({})
    await settle()

    expect(getRefreshed()).toBe(1)
  })

  it('does not tell when the backend answers after the relay is disposed', async () => {
    const { clock, relay, getRefreshed } = setup()
    relay.push({ paths: ['a.md'] })
    clock.fire()

    relay.dispose()
    await settle()

    expect(getRefreshed()).toBe(0)
  })
})
