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
})
