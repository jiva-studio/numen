import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { effectScope, shallowRef } from 'vue'
import { APPEAR_DELAY_MS, IDLE_CHIP, type ChipState } from '../model/chipDisplay'
import { useChipDisplay } from './useChipDisplay'

beforeEach(() => vi.useFakeTimers())
afterEach(() => vi.useRealTimers())

describe('useChipDisplay', () => {
  it('clears its timers when the scope ends', () => {
    const live = shallowRef<ChipState>(IDLE_CHIP)
    const scope = effectScope()
    const chip = scope.run(() => useChipDisplay(() => live.value))!

    live.value = { isActive: true, done: 0, total: 1, fraction: 0 }
    expect(vi.getTimerCount()).toBe(1)
    scope.stop()
    vi.advanceTimersByTime(APPEAR_DELAY_MS)

    expect(vi.getTimerCount()).toBe(0)
    expect(chip.value).toStrictEqual(IDLE_CHIP)
  })
})
