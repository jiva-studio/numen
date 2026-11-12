import { onScopeDispose, readonly, shallowRef, watch, type Ref } from 'vue'
import { getMonotonicNow } from '@numen/ui'
import { createChipDisplay, IDLE_CHIP, type ChipState } from '../model/chipDisplay'

/** The chip state to draw for the live one, delayed by createChipDisplay; its timers end with the scope. */
export function useChipDisplay(
  live: () => ChipState,
  now: () => number = getMonotonicNow,
): Readonly<Ref<ChipState>> {
  const chip = shallowRef<ChipState>(IDLE_CHIP)
  const display = createChipDisplay((state) => (chip.value = state), now)
  watch(live, display.update, { immediate: true, flush: 'sync' })
  onScopeDispose(display.stop)
  return readonly(chip)
}
