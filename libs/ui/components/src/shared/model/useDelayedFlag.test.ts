import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import { useDelayedFlag } from './useDelayedFlag'

const mountFlag = (delay: number) =>
  mount(
    defineComponent({
      setup() {
        const isOn = useDelayedFlag(() => delay)
        return () => h('p', String(isOn.value))
      },
    }),
  )

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe('a flag that turns on after a delay', () => {
  it('is on at once where the delay is none', () => {
    expect(mountFlag(0).text()).toBe('true')
  })

  it('turns on when the delay has passed and not before', async () => {
    const view = mountFlag(300)
    expect(view.text()).toBe('false')

    await vi.advanceTimersByTimeAsync(299)
    expect(view.text()).toBe('false')

    await vi.advanceTimersByTimeAsync(1)
    expect(view.text()).toBe('true')
  })

  it('stays off when the view is unmounted before the delay has passed', async () => {
    const view = mountFlag(300)
    view.unmount()

    await vi.advanceTimersByTimeAsync(300)

    expect(vi.getTimerCount()).toBe(0)
  })
})
