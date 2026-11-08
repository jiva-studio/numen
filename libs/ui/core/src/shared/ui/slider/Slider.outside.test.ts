/**
 * What the handle does when the value a caller gives it changes while the person is moving it: the
 * person's position stands, and the value from outside is taken once the person releases the handle.
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

const mountSlider = () => {
  const control = mount(Slider, {
    props: {
      modelValue: 1,
      min: 0.8,
      max: 2,
      step: 0.1,
      'onUpdate:modelValue': (value: number) => {
        control.setProps({ modelValue: value }).catch(() => undefined)
      },
    },
    attachTo: document.body,
  })
  return control
}

const getHandle = (control: ReturnType<typeof mountSlider>) => control.get('[role="slider"]')

const press = async (control: ReturnType<typeof mountSlider>, key: string) => {
  await getHandle(control).trigger('keydown', { key })
}

describe('a handle the person is moving', () => {
  it('stays where the person put it when the caller gives the value from before', async () => {
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

  it('stays where the person put it when the caller gives a value between', async () => {
    const control = mountSlider()
    await nextTick()
    await press(control, 'ArrowRight')
    await press(control, 'ArrowRight')

    await control.setProps({ modelValue: 1.1 })
    await nextTick()

    expect(getHandle(control).attributes('aria-valuenow')).toBe('1.2')
    control.unmount()
  })

  it('takes the value of the caller after the person releases the handle', async () => {
    const control = mountSlider()
    await nextTick()
    await press(control, 'ArrowRight')
    await getHandle(control).trigger('keyup', { key: 'ArrowRight' })

    await control.setProps({ modelValue: 1.6 })
    await nextTick()

    expect(getHandle(control).attributes('aria-valuenow')).toBe('1.6')
    control.unmount()
  })
})
