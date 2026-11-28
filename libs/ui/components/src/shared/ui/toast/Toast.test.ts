/**
 * What the toast shows, and when its text disappears.
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

const readStatus = (): string => document.body.querySelector('[role="status"]')?.textContent?.trim() ?? ''

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
    expect(readStatus()).toBe('Recovery kit saved')
  })

  it('shows nothing for an empty text', async () => {
    mountToast('')
    expect(readStatus()).toBe('')
    await vi.advanceTimersByTimeAsync(10_000)
    expect(readStatus()).toBe('')
  })

  it('hides its text once its duration has passed and not before', async () => {
    mountToast('Recovery kit saved', 3000)
    await vi.advanceTimersByTimeAsync(2999)
    expect(readStatus()).toBe('Recovery kit saved')
    await vi.advanceTimersByTimeAsync(1)
    expect(readStatus()).toBe('')
  })

  it('hides its text after 4 seconds when no duration is given', async () => {
    mountToast('Recovery kit saved')
    await vi.advanceTimersByTimeAsync(3999)
    expect(readStatus()).toBe('Recovery kit saved')
    await vi.advanceTimersByTimeAsync(1)
    expect(readStatus()).toBe('')
  })

  it('shows a new text for the whole duration again', async () => {
    const toast = mountToast('Recovery kit saved', 3000)
    await vi.advanceTimersByTimeAsync(2000)
    await toast.setProps({ text: 'Theme applied' })
    await vi.advanceTimersByTimeAsync(2000)
    expect(readStatus()).toBe('Theme applied')
    await vi.advanceTimersByTimeAsync(1000)
    expect(readStatus()).toBe('')
  })

  it('shows the same text again once it is given again', async () => {
    const toast = mountToast('Recovery kit saved', 3000)
    await vi.advanceTimersByTimeAsync(3000)
    await toast.setProps({ text: '' })
    await toast.setProps({ text: 'Recovery kit saved' })
    expect(readStatus()).toBe('Recovery kit saved')
  })
})
