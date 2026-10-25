import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h, ref, nextTick, type Ref } from 'vue'
import { usePageWidth, type PageWidthState } from './width'
import { WIDTH_DEBOUNCE, WIDTH_STEP, type Row } from '../lib/strip'

const createRow = (...widths: number[]): Row => ({
  height: 100,
  starts: widths.map((_, index) => index * 100),
  widths,
  length: widths.length * 100,
})

const setup = (...widths: number[]) => {
  const row: Ref<Row> = ref(createRow(...widths))
  const atWidth = vi.fn<(pixels: number) => void>()
  let state!: PageWidthState
  const wrapper = mount(
    defineComponent({
      setup() {
        state = usePageWidth(() => row.value, atWidth)
        return () => h('div')
      },
    }),
  )
  return { row, atWidth, state, wrapper }
}

describe('the width pages are requested for at', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('is the widest page, staged upward', () => {
    expect(setup(100, 200).state.needed.value).toBe(2 * WIDTH_STEP)
    expect(setup(WIDTH_STEP).state.needed.value).toBe(WIDTH_STEP)
    expect(setup(WIDTH_STEP + 1).state.needed.value).toBe(2 * WIDTH_STEP)
  })

  it('is nothing for a row with no pages', () => {
    expect(setup().state.needed.value).toBe(0)
  })

  it('scales by the device pixel ratio before staging', () => {
    vi.stubGlobal('devicePixelRatio', 2)
    expect(setup(100, 200).state.needed.value).toBe(4 * WIDTH_STEP)
  })

  it('takes a ratio that is not a number as one', () => {
    vi.stubGlobal('devicePixelRatio', Number.NaN)
    expect(setup(200).state.needed.value).toBe(2 * WIDTH_STEP)
  })

  it('takes one where there is no window', () => {
    const { row, state } = setup(200)
    vi.stubGlobal('window', undefined)
    row.value = createRow(300)
    const needed = state.needed.value
    vi.unstubAllGlobals()
    expect(needed).toBe(3 * WIDTH_STEP)
  })

  it('asks at once for the first width', async () => {
    const { row, atWidth, state } = setup()
    row.value = createRow(200)
    await nextTick()
    expect(atWidth).toHaveBeenCalledExactlyOnceWith(2 * WIDTH_STEP)
    expect(state.drawnWidth.value).toBe(2 * WIDTH_STEP)
  })

  it('asks for a changed width only once it has stood still', async () => {
    const { row, atWidth, state } = setup(200)
    row.value = createRow(300)
    await nextTick()
    expect(atWidth).not.toHaveBeenCalled()
    expect(state.drawnWidth.value).toBe(0)
    vi.advanceTimersByTime(WIDTH_DEBOUNCE)
    expect(atWidth).toHaveBeenCalledExactlyOnceWith(3 * WIDTH_STEP)
    expect(state.drawnWidth.value).toBe(3 * WIDTH_STEP)
  })

  it('asks only for the last of widths that follow one another', async () => {
    const { row, atWidth } = setup(200)
    row.value = createRow(300)
    await nextTick()
    vi.advanceTimersByTime(WIDTH_DEBOUNCE - 1)
    row.value = createRow(500)
    await nextTick()
    vi.advanceTimersByTime(WIDTH_DEBOUNCE - 1)
    expect(atWidth).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(atWidth).toHaveBeenCalledExactlyOnceWith(4 * WIDTH_STEP)
  })

  it('does not ask for no width', async () => {
    const { row, atWidth } = setup(200)
    row.value = createRow()
    await nextTick()
    vi.advanceTimersByTime(WIDTH_DEBOUNCE * 2)
    expect(atWidth).not.toHaveBeenCalled()
  })

  it('does not ask after it was unmounted', async () => {
    const { row, atWidth, wrapper } = setup(200)
    row.value = createRow(300)
    await nextTick()
    wrapper.unmount()
    vi.advanceTimersByTime(WIDTH_DEBOUNCE * 2)
    expect(atWidth).not.toHaveBeenCalled()
  })
})
