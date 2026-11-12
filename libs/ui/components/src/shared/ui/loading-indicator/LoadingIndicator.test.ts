/**
 * When the ring is drawn: never before the delay, so what arrives at once
 * replaces an empty pane without a flash.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import LoadingIndicator from './LoadingIndicator.vue'

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('a pane that is loading', () => {
  it('is announced as busy and as a status, from the first moment', () => {
    const pane = mount(LoadingIndicator)
    expect(pane.attributes('role')).toBe('status')
    expect(pane.attributes('aria-busy')).toBe('true')
  })

  it('draws nothing until the delay has passed', async () => {
    const pane = mount(LoadingIndicator, { props: { delay: 400 } })
    expect(pane.attributes('data-shown')).toBe('false')
    expect(pane.find('.spinner').exists()).toBe(false)

    await vi.advanceTimersByTimeAsync(399)
    expect(pane.find('.spinner').exists()).toBe(false)

    await vi.advanceTimersByTimeAsync(1)
    expect(pane.attributes('data-shown')).toBe('true')
    expect(pane.find('.spinner').exists()).toBe(true)
    expect(pane.text()).toBe('Loading…')
  })

  it('draws at once where the delay is none', () => {
    const pane = mount(LoadingIndicator, { props: { delay: 0 } })
    expect(pane.find('.spinner').exists()).toBe(true)
    expect(vi.getTimerCount()).toBe(0)
  })

  it('writes the words it is given', () => {
    const pane = mount(LoadingIndicator, { props: { delay: 0, label: 'Reading the shed' } })
    expect(pane.text()).toBe('Reading the shed')
  })

  it('takes its timer away with it, so nothing is drawn into a pane that has gone', () => {
    const pane = mount(LoadingIndicator, { props: { delay: 400 } })
    expect(vi.getTimerCount()).toBe(1)
    pane.unmount()
    expect(vi.getTimerCount()).toBe(0)
  })
})
