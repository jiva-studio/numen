/**
 * What the handle is announced as, and what it hands on when it is moved.
 *
 * The control carries one value, so what a caller gives it and what it hands
 * back is a number and never a list.
 */
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Slider from './Slider.vue'

// The track is measured as it is drawn, and a document with no layout in it
// watches nothing for size.
beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
})

type SliderProps = InstanceType<typeof Slider>['$props']

const mountSlider = (props: Partial<SliderProps> = {}) =>
  mount(Slider, { props: { modelValue: 40, ...props } })

/** The handle, which is the thing the keyboard moves. */
const handle = (control: ReturnType<typeof mountSlider>) => control.get('[role="slider"]')

/** Every value the control has handed on, in the order it handed them on. */
const handed = (control: ReturnType<typeof mountSlider>): readonly unknown[] =>
  (control.emitted('update:modelValue') ?? []).map((said) => (said as unknown[])[0])

describe('what a screen reader is told', () => {
  it('is one handle standing at a value, between the two ends', async () => {
    const control = mountSlider()
    await nextTick()
    expect(control.findAll('[role="slider"]')).toHaveLength(1)
    expect(handle(control).attributes('aria-valuenow')).toBe('40')
    expect(handle(control).attributes('aria-valuemin')).toBe('0')
    expect(handle(control).attributes('aria-valuemax')).toBe('100')
    expect(handle(control).attributes('aria-orientation')).toBe('horizontal')
  })

  it('runs between the ends it is given', async () => {
    const control = mountSlider({ min: 10, max: 20, modelValue: 12 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('12')
    expect(handle(control).attributes('aria-valuemin')).toBe('10')
    expect(handle(control).attributes('aria-valuemax')).toBe('20')
  })

  // The handle is what carries the role, so it is what the words beside the
  // control have to name.
  it('is named by the words a caller points it at', () => {
    const control = mountSlider({ 'aria-labelledby': 'said' } as Partial<SliderProps>)
    expect(handle(control).attributes('aria-labelledby')).toBe('said')
    expect(control.get('[data-slot="slider"]').attributes('aria-labelledby')).toBeUndefined()
  })

  it('is the stop on the way round the screen, and the track is none', () => {
    const control = mountSlider()
    expect(handle(control).attributes('tabindex')).toBe('0')
    expect(control.get('[data-slot="slider"]').attributes('tabindex')).toBeUndefined()
  })
})

describe('moving the handle', () => {
  it('hands back a number and not a list', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    expect(handed(control)).toEqual([41])
  })

  // A caller puts back what it was handed, which is what a walk of the track
  // is: each step is taken from where the last one left it.
  it('moves a step at a time, by the step it is given', async () => {
    const control = mountSlider({ step: 5 })
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    await control.setProps({ modelValue: 45 })
    await handle(control).trigger('keydown', { key: 'ArrowLeft' })
    expect(handed(control)).toEqual([45, 40])
  })

  it('goes to either end, and no further', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'Home' })
    await handle(control).trigger('keydown', { key: 'End' })
    expect(handed(control)).toEqual([0, 100])

    const least = mountSlider({ modelValue: 0 })
    await handle(least).trigger('keydown', { key: 'ArrowLeft' })
    expect(handed(least)).toEqual([])
  })

  it('hands nothing on while nobody may move it', async () => {
    const control = mountSlider({ disabled: true })
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    expect(handed(control)).toEqual([])
  })
})

describe('a value the ends do not hold', () => {
  it('stands at the end it is past, and is announced there', async () => {
    const control = mountSlider({ modelValue: 90, max: 50 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('50')
    expect(handed(control)).toEqual([50])
  })

  it('stands at the floor where it is handed a value under it', async () => {
    const control = mountSlider({ modelValue: -20 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('0')
    expect(handed(control)).toEqual([0])
  })

  // A caller narrowing what a control allows narrows it under a value already
  // standing there, and what is announced is where the handle is drawn.
  it('comes inside the ends where they move under it', async () => {
    const control = mountSlider({ modelValue: 90 })
    await control.setProps({ max: 50 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('50')
    expect(handed(control)).toEqual([50])
  })
})

describe('the box it is drawn in', () => {
  it('is the caller’s to size, and the handle is left alone', () => {
    const control = mountSlider({ class: 'w-40' })
    expect(control.get('[data-slot="slider"]').classes()).toContain('w-40')
    expect(handle(control).classes()).not.toContain('w-40')
  })
})

describe('coming to rest', () => {
  // A caller writes what it was handed when the handle settles, so the value
  // has to be handed on before the settling is.
  it('hands the value on before it says the handle has settled', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'ArrowRight' })

    expect(handed(control)).toEqual([41])
    expect(control.emitted('settles')).toEqual([[41]])
    const order = Object.keys(control.emitted())
    expect(order.indexOf('update:modelValue')).toBeLessThan(order.indexOf('settles'))
  })

  it('says nothing about settling while the handle is only moving', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'Home' })
    await control.setProps({ modelValue: 0 })
    expect(control.emitted('settles')).toEqual([[0]])
  })
})
