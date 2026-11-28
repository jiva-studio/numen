import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { useLinkScan } from './useLinkScan'

const mountScan = (findCamera: () => Promise<boolean>) => {
  let scan!: ReturnType<typeof useLinkScan>
  const wrapper = mount(
    defineComponent({
      setup() {
        scan = useLinkScan(findCamera)
        return () => h('div')
      },
    }),
  )
  return { scan, wrapper }
}

describe('useLinkScan', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('shows the camera when one is found', async () => {
    const { scan } = mountScan(() => Promise.resolve(true))
    await vi.advanceTimersByTimeAsync(0)
    expect(scan.phase.value).toBe('camera')
  })

  it('shows the paste field when the check takes longer than two seconds', async () => {
    const { scan } = mountScan(() => new Promise<boolean>(() => undefined))
    await vi.advanceTimersByTimeAsync(2000)
    expect(scan.phase.value).toBe('paste')
  })

  it('leaves no timer and keeps its phase when the form closes during the check', async () => {
    const { scan, wrapper } = mountScan(() => new Promise<boolean>(() => undefined))
    wrapper.unmount()
    await vi.advanceTimersByTimeAsync(0)
    expect(vi.getTimerCount()).toBe(0)
    expect(scan.phase.value).toBe('checking')
  })
})
