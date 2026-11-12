import { ref, watch, type Ref, type WatchSource } from 'vue'

const POPOVER_GAP = 4

/**
 * The side offset of a popover that keeps `POPOVER_GAP` px from the edge of the bar, whatever the
 * height of the chip. floating-ui measures the offset from the chip, which is centered in
 * the bar. The offset is read from the bar tokens each time `isOpen` turns true.
 */
export function useSideOffset(isOpen: WatchSource<boolean>): Ref<number> {
  const sideOffset = ref(POPOVER_GAP)

  function getSideOffset(): number {
    const style = getComputedStyle(document.documentElement)
    const barHeight = Number.parseFloat(style.getPropertyValue('--numen-title-bar-height'))
    const chipHeight = Number.parseFloat(style.getPropertyValue('--numen-chip-height'))
    if (Number.isNaN(barHeight) || Number.isNaN(chipHeight)) return POPOVER_GAP
    return (barHeight - chipHeight) / 2 + POPOVER_GAP
  }

  watch(
    isOpen,
    (value) => {
      if (value) sideOffset.value = getSideOffset()
    },
    { immediate: true, flush: 'sync' },
  )

  return sideOffset
}
