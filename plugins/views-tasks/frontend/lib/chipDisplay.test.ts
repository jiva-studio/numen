import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  APPEAR_DELAY_MS,
  createChipDisplay,
  HOLD_MS,
  IDLE_CHIP,
  type ChipState,
} from './chipDisplay'

const chip = (done: number, total: number): ChipState => ({
  isActive: true,
  done,
  total,
  fraction: done / total,
})

function setup() {
  const output = { value: IDLE_CHIP }
  const display = createChipDisplay(
    (state) => (output.value = state),
    () => Date.now(),
  )
  const live = {
    set value(state: ChipState) {
      display.update(state)
    },
  }
  return { live, displayed: output, display }
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('createChipDisplay', () => {
  it('never shows a task shorter than the appearance delay', () => {
    const { live, displayed } = setup()

    live.value = chip(0, 1)
    vi.advanceTimersByTime(APPEAR_DELAY_MS - 1)
    live.value = IDLE_CHIP
    vi.advanceTimersByTime(HOLD_MS * 2)

    expect(displayed.value).toStrictEqual(IDLE_CHIP)
  })

  it('never shows a chain of short tasks that ends before the appearance delay', () => {
    const { live, displayed } = setup()
    const frames: ChipState[] = []

    live.value = chip(0, 1)
    vi.advanceTimersByTime(100)
    live.value = chip(1, 2)
    frames.push(displayed.value)
    vi.advanceTimersByTime(100)
    live.value = chip(2, 3)
    frames.push(displayed.value)
    vi.advanceTimersByTime(99)
    live.value = IDLE_CHIP
    vi.advanceTimersByTime(HOLD_MS * 2)

    expect(frames).toStrictEqual([IDLE_CHIP, IDLE_CHIP])
    expect(displayed.value).toStrictEqual(IDLE_CHIP)
  })

  it('shows the latest state once a chain of tasks has lasted the appearance delay', () => {
    const { live, displayed } = setup()

    live.value = chip(0, 1)
    vi.advanceTimersByTime(200)
    live.value = chip(1, 2)
    vi.advanceTimersByTime(APPEAR_DELAY_MS - 200)

    expect(displayed.value).toStrictEqual(chip(1, 2))
  })

  it('delays changes closer than the hold time and shows only the latest', () => {
    const { live, displayed } = setup()
    live.value = chip(0, 3)
    vi.advanceTimersByTime(APPEAR_DELAY_MS)

    vi.advanceTimersByTime(100)
    live.value = chip(1, 3)
    vi.advanceTimersByTime(100)
    live.value = chip(2, 3)

    expect(displayed.value).toStrictEqual(chip(0, 3))
    vi.advanceTimersByTime(HOLD_MS - 200)
    expect(displayed.value).toStrictEqual(chip(2, 3))
  })

  it('shows a change at once when the hold time has already passed', () => {
    const { live, displayed } = setup()
    live.value = chip(0, 3)
    vi.advanceTimersByTime(APPEAR_DELAY_MS + HOLD_MS)

    live.value = chip(1, 3)

    expect(displayed.value).toStrictEqual(chip(1, 3))
  })

  it('delays the return to idle', () => {
    const { live, displayed } = setup()
    live.value = chip(0, 1)
    vi.advanceTimersByTime(APPEAR_DELAY_MS)

    vi.advanceTimersByTime(50)
    live.value = IDLE_CHIP
    expect(displayed.value).toStrictEqual(chip(0, 1))

    vi.advanceTimersByTime(HOLD_MS - 51)
    expect(displayed.value).toStrictEqual(chip(0, 1))
    vi.advanceTimersByTime(1)
    expect(displayed.value).toStrictEqual(IDLE_CHIP)
  })

  it('keeps the shown state when a delayed change is undone before the hold ends', () => {
    const { live, displayed } = setup()
    live.value = chip(0, 3)
    vi.advanceTimersByTime(APPEAR_DELAY_MS)

    live.value = chip(1, 3)
    live.value = chip(0, 3)
    vi.advanceTimersByTime(HOLD_MS * 2)

    expect(displayed.value).toStrictEqual(chip(0, 3))
  })

  it('clears its timers on stop', () => {
    const { live, displayed, display } = setup()

    live.value = chip(0, 1)
    expect(vi.getTimerCount()).toBe(1)
    display.stop()
    vi.advanceTimersByTime(APPEAR_DELAY_MS)

    expect(vi.getTimerCount()).toBe(0)
    expect(displayed.value).toStrictEqual(IDLE_CHIP)
  })

  it('clears a pending delay timer on stop', () => {
    const { live, displayed, display } = setup()
    live.value = chip(0, 3)
    vi.advanceTimersByTime(APPEAR_DELAY_MS)
    live.value = chip(1, 3)
    expect(vi.getTimerCount()).toBe(1)

    display.stop()

    expect(vi.getTimerCount()).toBe(0)
    expect(displayed.value).toStrictEqual(chip(0, 3))
  })
})
