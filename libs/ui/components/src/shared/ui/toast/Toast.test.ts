/**
 * What the toast shows, and when it asks to be closed.
 */
import { mount, type VueWrapper } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Toast from './Toast.vue'

const mounted: VueWrapper[] = []

const mountToast = (text: string, durationMs?: number) => {
  const toast = mount(Toast, {
    props: durationMs === undefined ? { text } : { text, durationMs },
  })
  mounted.push(toast)
  return toast
}

const readStatus = (): string => document.body.querySelector('[role="status"]')?.textContent ?? ''

const countCloses = (toast: VueWrapper): number => toast.emitted('close')?.length ?? 0

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  while (mounted.length) mounted.pop()?.unmount()
  vi.useRealTimers()
})

describe('Toast', () => {
  it('shows its text in a live status region', () => {
    mountToast('Recovery kit saved')
    expect(readStatus().trim()).toBe('Recovery kit saved')
  })

  it('shows nothing for an empty text and never asks to be closed', async () => {
    const toast = mountToast('')
    expect(readStatus().trim()).toBe('')
    await vi.advanceTimersByTimeAsync(10_000)
    expect(countCloses(toast)).toBe(0)
  })

  it('asks to be closed once its duration has passed and not before', async () => {
    const toast = mountToast('Recovery kit saved', 3000)
    await vi.advanceTimersByTimeAsync(2999)
    expect(countCloses(toast)).toBe(0)
    await vi.advanceTimersByTimeAsync(1)
    expect(countCloses(toast)).toBe(1)
  })

  it('shows a new text for the whole duration again', async () => {
    const toast = mountToast('Recovery kit saved', 3000)
    await vi.advanceTimersByTimeAsync(2000)
    await toast.setProps({ text: 'Theme applied' })
    await vi.advanceTimersByTimeAsync(2000)
    expect(countCloses(toast)).toBe(0)
    expect(readStatus().trim()).toBe('Theme applied')
    await vi.advanceTimersByTimeAsync(1000)
    expect(countCloses(toast)).toBe(1)
  })

  it('does not ask to be closed after it is unmounted', async () => {
    const toast = mountToast('Recovery kit saved', 3000)
    const closes = toast.emitted()
    toast.unmount()
    mounted.pop()
    await vi.advanceTimersByTimeAsync(5000)
    expect(closes['close']).toBeUndefined()
  })
})
