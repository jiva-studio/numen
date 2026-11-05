/**
 * What the handle does when the value a caller gives it changes while the person is moving it: the
 * person's position stands, and the value from outside is taken once the person lets go.
 */
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Slider from './Slider.vue'

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

const mountSlider = () =>
  mount(Slider, { props: { modelValue: 1, min: 0.8, max: 2, step: 0.1 }, attachTo: document.body })

const getHandle = (control: ReturnType<typeof mountSlider>) => control.get('[role="slider"]')

const press = async (control: ReturnType<typeof mountSlider>, key: string) => {
  await getHandle(control).trigger('keydown', { key })
}

describe('a handle the person is moving', () => {
  it('stays where the person put it when the caller gives the old value again', async () => {
    const control = mountSlider()
    await nextTick()

    await press(control, 'ArrowRight')
    await press(control, 'ArrowRight')
    expect(getHandle(control).attributes('aria-valuenow')).toBe('1.2')

    await control.setProps({ modelValue: 1 })
    await nextTick()

    expect(getHandle(control).attributes('aria-valuenow')).toBe('1.2')
    control.unmount()
  })

  it('does not hand the old value on when the caller gives it again', async () => {
    const control = mountSlider()
    await nextTick()
    await press(control, 'ArrowRight')
    await press(control, 'ArrowRight')

    await control.setProps({ modelValue: 1.1 })
    await nextTick()

    const handed = (control.emitted('update:modelValue') ?? []).map((one) => (one as number[])[0])
    expect(handed[handed.length - 1]).toBeCloseTo(1.2)
    control.unmount()
  })

  it('takes the value of the caller once the person lets go', async () => {
    const control = mountSlider()
    await nextTick()
    await press(control, 'ArrowRight')
    await control.setProps({ modelValue: 1.5 })
    await getHandle(control).trigger('keyup', { key: 'ArrowRight' })
    await nextTick()

    expect(getHandle(control).attributes('aria-valuenow')).toBe('1.5')
    control.unmount()
  })
})
