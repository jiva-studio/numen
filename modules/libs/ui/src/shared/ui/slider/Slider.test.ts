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
const getEmitted = (control: ReturnType<typeof mountSlider>): readonly unknown[] =>
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
    expect(getEmitted(control)).toEqual([41])
  })

  // A caller puts back what it was handed, which is what a walk of the track
  // is: each step is taken from where the last one left it.
  it('moves a step at a time, by the step it is given', async () => {
    const control = mountSlider({ step: 5 })
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    await control.setProps({ modelValue: 45 })
    await handle(control).trigger('keydown', { key: 'ArrowLeft' })
    expect(getEmitted(control)).toEqual([45, 40])
  })

  it('goes to either end, and no further', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'Home' })
    await handle(control).trigger('keydown', { key: 'End' })
    expect(getEmitted(control)).toEqual([0, 100])

    const least = mountSlider({ modelValue: 0 })
    await handle(least).trigger('keydown', { key: 'ArrowLeft' })
    expect(getEmitted(least)).toEqual([])
  })

  // A step out and a step back come to where they began, and the ceiling is a
  // place the handle may stand whether or not the step lays one there.
  it('gives back at the ceiling what a step took to get there', async () => {
    const control = mountSlider({ modelValue: 9, min: 0, max: 10, step: 3 })
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    await control.setProps({ modelValue: 10 })
    await handle(control).trigger('keydown', { key: 'ArrowLeft' })

    expect(getEmitted(control)).toEqual([10, 9])
  })

  it('moves ten steps under a page key, and under a key held with shift', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'PageUp' })
    await control.setProps({ modelValue: 50 })
    await handle(control).trigger('keydown', { key: 'PageDown' })
    await control.setProps({ modelValue: 40 })
    await handle(control).trigger('keydown', { key: 'ArrowRight', shiftKey: true })

    expect(getEmitted(control)).toEqual([50, 40, 50])
  })

  it('hands nothing on while nobody may move it', async () => {
    const control = mountSlider({ disabled: true })
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    expect(getEmitted(control)).toEqual([])
  })
})

describe('a value the ends do not hold', () => {
  it('stands at the end it is past, and is announced there', async () => {
    const control = mountSlider({ modelValue: 90, max: 50 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('50')
    expect(getEmitted(control)).toEqual([50])
  })

  it('stands at the floor where it is handed a value under it', async () => {
    const control = mountSlider({ modelValue: -20 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('0')
    expect(getEmitted(control)).toEqual([0])
  })

  // A caller narrowing what a control allows narrows it under a value already
  // standing there, and what is announced is where the handle is drawn.
  it('comes inside the ends where they move under it', async () => {
    const control = mountSlider({ modelValue: 90 })
    await control.setProps({ max: 50 })
    await nextTick()
    expect(handle(control).attributes('aria-valuenow')).toBe('50')
    expect(getEmitted(control)).toEqual([50])
  })
})

describe('coming to rest', () => {
  /** A walk of so many places, the key held down the whole way. */
  const walkTrack = async (control: ReturnType<typeof mountSlider>, places: readonly number[]) => {
    for (const at of places) {
      await handle(control).trigger('keydown', { key: 'ArrowRight' })
      await control.setProps({ modelValue: at })
    }
  }

  // A caller writes what it was handed when the handle settles, so the value
  // has to be handed on before the settling is.
  it('hands the value on before it says the handle has settled', async () => {
    const control = mountSlider()
    await handle(control).trigger('keydown', { key: 'ArrowRight' })
    await handle(control).trigger('keyup', { key: 'ArrowRight' })

    expect(getEmitted(control)).toEqual([41])
    expect(control.emitted('settles')).toEqual([[41]])
    const order = Object.keys(control.emitted())
    expect(order.indexOf('update:modelValue')).toBeLessThan(order.indexOf('settles'))
  })

  it('says nothing about settling while the keys are still walking it', async () => {
    const control = mountSlider()
    await walkTrack(control, [41, 42, 43])

    expect(getEmitted(control)).toEqual([41, 42, 43])
    expect(control.emitted('settles')).toBeUndefined()
  })

  it('says it once, at where the walk left it, when the key is let go of', async () => {
    const control = mountSlider()
    await walkTrack(control, [41, 42, 43])
    await handle(control).trigger('keyup', { key: 'ArrowRight' })

    expect(control.emitted('settles')).toEqual([[43]])
  })

  it('says nothing where the walk left the handle where it began', async () => {
    const control = mountSlider({ modelValue: 0 })
    await handle(control).trigger('keydown', { key: 'ArrowLeft' })
    await handle(control).trigger('keyup', { key: 'ArrowLeft' })

    expect(control.emitted('settles')).toBeUndefined()
  })

  // A handle the keyboard leaves in the middle of a walk is a handle let go of.
  it('says it where the keyboard leaves the handle with the key still down', async () => {
    const control = mountSlider()
    await walkTrack(control, [41])
    await handle(control).trigger('blur')

    expect(control.emitted('settles')).toEqual([[41]])
  })
})
