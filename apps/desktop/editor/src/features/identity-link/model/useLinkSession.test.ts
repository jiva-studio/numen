/** The session of a link reads the link the core holds and passes the answers of the person on, against the real store. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { createIdentityStore, type LinkRole } from '@/entities/identity'
import {
  createIdentityClient,
  createLinkInfo,
  type IdentityCalls,
  wrapIdentityClient,
} from '@/testing/identityClient'
import { useLinkSession } from './useLinkSession'

const mountSession = (overrides: Partial<IdentityCalls>, role: LinkRole = 'joiner') => {
  const store = createIdentityStore(wrapIdentityClient(createIdentityClient(overrides)), vi.fn())
  let session!: ReturnType<typeof useLinkSession>
  const wrapper = mount(
    defineComponent({
      setup() {
        session = useLinkSession(() => store.devices.link, role)
        return () => h('div')
      },
    }),
  )
  return { session, answers: session.answers, wrapper }
}

const ok = <T>(value: T) => Promise.resolve({ ok: true as const, value })

describe('useLinkSession', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('has no link before a mode is chosen', () => {
    const { session } = mountSession({})
    expect(session.link.value).toBeUndefined()
  })

  it('starts to show the QR code of a host when it opens', async () => {
    const startLink = vi.fn(() => ok(createLinkInfo()))
    mountSession({ startLink }, 'host')
    await vi.advanceTimersByTimeAsync(0)
    expect(startLink).toHaveBeenCalledWith('host', 'show', '')
  })

  it('reports a link that waits 15 seconds for its channel as stalled and cancels it', async () => {
    const waiting = createLinkInfo({ phase: 'waitingForChannel', qrText: '' })
    const cancelLink = vi.fn(() => ok(undefined))
    const { session } = mountSession({
      startLink: () => ok(waiting),
      getLink: () => ok(waiting),
      cancelLink,
    })
    await session.start('show')
    await vi.advanceTimersByTimeAsync(14000)
    expect(session.isStalled.value).toBe(false)
    await vi.advanceTimersByTimeAsync(1500)
    expect(session.isStalled.value).toBe(true)
    expect(cancelLink).toHaveBeenCalledOnce()
  })

  it('starts the same link again on retry', async () => {
    const startLink = vi.fn(() => ok(createLinkInfo()))
    const { session } = mountSession({ startLink, getLink: () => ok(createLinkInfo()) })
    await session.start('scan', 'numen://x')
    await session.retry()
    expect(startLink).toHaveBeenLastCalledWith('joiner', 'scan', 'numen://x')
  })

  it('starts a link and reads it again every half second until it ends', async () => {
    const getLink = vi.fn(() => ok(createLinkInfo()))
    const { session } = mountSession({ startLink: () => ok(createLinkInfo()), getLink })
    await session.start('show')
    await vi.advanceTimersByTimeAsync(1500)
    expect(getLink).toHaveBeenCalledTimes(3)
    expect(session.link.value?.qrText).toBe(createLinkInfo().qrText)
  })

  it('stops reading once the link failed', async () => {
    const failed = createLinkInfo({ phase: 'failed', reason: 'declined' })
    const getLink = vi.fn(() => ok(failed))
    const { session } = mountSession({ startLink: () => ok(createLinkInfo()), getLink })
    await session.start('show')
    await vi.advanceTimersByTimeAsync(2000)
    expect(getLink).toHaveBeenCalledTimes(1)
  })

  it('shows the refusal of a start and holds no link', async () => {
    const { session } = mountSession({
      startLink: () =>
        Promise.resolve({ ok: false, error: { error: 'linkTextInvalid', message: 'x' } }),
    })
    await session.start('scan', 'x')
    expect(session.problem.value).toBe('This text is not the link of a QR code.')
  })

  it('cancels an active link when the dialog closes', async () => {
    const cancelLink = vi.fn(() => ok(undefined))
    const { session, wrapper } = mountSession({
      startLink: () => ok(createLinkInfo()),
      getLink: () => ok(createLinkInfo()),
      cancelLink,
    })
    await session.start('show')
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(0)
    expect(cancelLink).toHaveBeenCalledOnce()
  })

  it('passes the answer of the person on and reads the link again', async () => {
    const decideLink = vi.fn(() => ok(undefined))
    const getLink = vi.fn(() => ok(createLinkInfo({ phase: 'confirmPeer' })))
    const { session, answers } = mountSession({
      startLink: () => ok(createLinkInfo()),
      getLink,
      decideLink,
    })
    await session.start('show')
    await answers.decide(true)
    expect(decideLink).toHaveBeenCalledWith(true)
    expect(getLink).toHaveBeenCalled()
  })
  it('leaves no timer and cancels the link when the dialog closes while the link starts', async () => {
    let finish: (value: ReturnType<typeof createLinkInfo>) => void = () => undefined
    const cancelLink = vi.fn(() => ok(undefined))
    const getLink = vi.fn(() => ok(createLinkInfo()))
    const { session, wrapper } = mountSession({
      startLink: () =>
        new Promise((resolve) => (finish = (value) => resolve({ ok: true as const, value }))),
      getLink,
      cancelLink,
    })
    const starting = session.start('show')
    await vi.advanceTimersByTimeAsync(0)
    wrapper.unmount()
    finish(createLinkInfo())
    await starting
    await vi.advanceTimersByTimeAsync(2000)
    expect(getLink).not.toHaveBeenCalled()
    expect(cancelLink).toHaveBeenCalled()
  })

  it('does not stay busy when a call throws', async () => {
    const { session } = mountSession({
      startLink: () => Promise.reject(new Error('down')),
    })
    await session.start('show').catch(() => undefined)
    expect(session.isBusy.value).toBe(false)
  })

  it('does not stay busy when an answer throws', async () => {
    const { session, answers } = mountSession({
      startLink: () => ok(createLinkInfo()),
      getLink: () => ok(createLinkInfo()),
      decideLink: () => Promise.reject(new Error('down')),
    })
    await session.start('show')
    await answers.decide(true).catch(() => undefined)
    expect(session.isBusy.value).toBe(false)
  })

  it('keeps one timer when it starts twice', async () => {
    const getLink = vi.fn(() => ok(createLinkInfo()))
    const { session } = mountSession({ startLink: () => ok(createLinkInfo()), getLink })
    await Promise.all([session.start('show'), session.start('show')])
    await vi.advanceTimersByTimeAsync(1000)
    expect(getLink).toHaveBeenCalledTimes(2)
  })

  it('drops an answer to an older read when a newer one came', async () => {
    const resolvers: Array<(info: ReturnType<typeof createLinkInfo>) => void> = []
    const getLink = vi.fn(
      () =>
        new Promise<{ ok: true; value: ReturnType<typeof createLinkInfo> }>((resolve) =>
          resolvers.push((value) => resolve({ ok: true as const, value })),
        ),
    )
    const { session } = mountSession({ startLink: () => ok(createLinkInfo()), getLink })
    await session.start('show')
    await vi.advanceTimersByTimeAsync(1000)
    resolvers[1]?.(createLinkInfo({ phase: 'confirmPeer' }))
    await vi.advanceTimersByTimeAsync(0)
    resolvers[0]?.(createLinkInfo({ phase: 'waitingForChannel' }))
    await vi.advanceTimersByTimeAsync(0)
    expect(session.link.value?.phase).toBe('confirmPeer')
  })

  it('reports the link as stalled and stops reading after the reads keep failing', async () => {
    const getLink = vi.fn(() => Promise.reject<never>(new Error('down')))
    const { session } = mountSession({ startLink: () => ok(createLinkInfo()), getLink })
    await session.start('show')
    await vi.advanceTimersByTimeAsync(10_000)
    expect(session.isStalled.value).toBe(true)
    const reads = getLink.mock.calls.length
    await vi.advanceTimersByTimeAsync(5000)
    expect(getLink.mock.calls.length).toBe(reads)
  })
})
