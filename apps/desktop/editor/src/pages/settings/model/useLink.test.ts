/** The dialog of a link reads the link the core holds and passes the answers of the person on. */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import type { LinkInfo } from '@/entities/identity'
import type { InstallationLink } from '../types'
import { useLink } from './useLink'

const LINK: LinkInfo = {
  phase: 'waiting_for_peer',
  mode: 'show',
  role: 'host',
  qrText: 'numen://link?v=1',
  code: '',
  peerName: '',
  peerFingerprint: '',
  accountFingerprint: 'AAAA',
  triesLeft: 3,
  reason: undefined,
  deviceName: 'Laptop',
  deviceFingerprint: 'BBBB',
  isApproved: false,
}

const createLink = (overrides: Partial<InstallationLink> = {}): InstallationLink => ({
  start: () => Promise.resolve({ ok: true, value: LINK }),
  get: () => Promise.resolve({ ok: true, value: LINK }),
  decide: () => Promise.resolve({ ok: true, value: undefined }),
  enterCode: () => Promise.resolve({ ok: true, value: undefined }),
  confirmJoin: () => Promise.resolve({ ok: true, value: undefined }),
  cancel: () => Promise.resolve({ ok: true, value: undefined }),
  ...overrides,
})

const mountLink = (link: InstallationLink, getBlock?: () => string) => {
  let state!: ReturnType<typeof useLink>
  const wrapper = mount(
    defineComponent({
      setup() {
        state = useLink(() => link, 'host', getBlock)
        return () => h('div')
      },
    }),
  )
  return { state, wrapper }
}

describe('useLink', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('has no link before a mode is chosen', () => {
    const { state } = mountLink(createLink())
    expect(state.link.value).toBeUndefined()
    expect(state.isChoosing.value).toBe(true)
  })

  it('starts a link and reads it again every half second until it ends', async () => {
    const get = vi.fn(() => Promise.resolve({ ok: true as const, value: LINK }))
    const { state } = mountLink(createLink({ get }))
    await state.start('show')
    await vi.advanceTimersByTimeAsync(1500)
    expect(get).toHaveBeenCalledTimes(3)
    expect(state.link.value?.qrText).toBe('numen://link?v=1')
  })

  it('stops reading once the link failed', async () => {
    const failed: LinkInfo = { ...LINK, phase: 'failed', reason: 'declined' }
    const get = vi.fn(() => Promise.resolve({ ok: true as const, value: failed }))
    const { state } = mountLink(createLink({ get }))
    await state.start('show')
    await vi.advanceTimersByTimeAsync(2000)
    expect(get).toHaveBeenCalledTimes(1)
  })

  it('shows the refusal of a start and holds no link', async () => {
    const { state } = mountLink(
      createLink({ start: () => Promise.resolve({ ok: false, error: 'Not a link.' }) }),
    )
    await state.start('scan', 'x')
    expect(state.problem.value).toBe('Not a link.')
    expect(state.isChoosing.value).toBe(true)
  })

  it('starts nothing where the device is blocked, and says why', async () => {
    const start = vi.fn(() => Promise.resolve({ ok: true as const, value: LINK }))
    const { state } = mountLink(createLink({ start }), () => 'This device has an identity.')
    await state.start('show')
    expect(start).not.toHaveBeenCalled()
    expect(state.problem.value).toBe('This device has an identity.')
  })

  it('cancels a link that is still going when the dialog goes away', async () => {
    const cancel = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    const { state, wrapper } = mountLink(createLink({ cancel }))
    await state.start('show')
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(0)
    expect(cancel).toHaveBeenCalledOnce()
  })

  it('passes the answer of the person on and reads the link again', async () => {
    const decide = vi.fn(() => Promise.resolve({ ok: true as const, value: undefined }))
    const get = vi.fn(() => Promise.resolve({ ok: true as const, value: LINK }))
    const { state } = mountLink(createLink({ decide, get }))
    await state.start('show')
    await state.decide(true)
    expect(decide).toHaveBeenCalledWith(true)
    expect(get).toHaveBeenCalled()
  })
})
