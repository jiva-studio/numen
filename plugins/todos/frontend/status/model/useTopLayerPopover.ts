/**
 * Places a popover in the browser top layer under the right edge of its trigger,
 * where no scroll bar or stacking context of the page can paint over it.
 */
import { measureRect, measureWindow, win } from '@numen/ui'
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import type { Ref, ShallowRef } from 'vue'

export function showInTopLayer(el: HTMLElement | null | undefined): void {
  if (el && typeof el.showPopover === 'function') el.showPopover()
}

export function useTopLayerPopover(
  isOpen: Readonly<Ref<boolean>>,
  trigger: Readonly<ShallowRef<HTMLElement | null>>,
  popover: Readonly<ShallowRef<HTMLElement | null>>,
) {
  const style = ref<Record<string, string>>({})

  watch(isOpen, (open) => {
    win.removeEventListener('resize', place)
    if (!open) return
    win.addEventListener('resize', place)
    place()
    nextTick(() => showInTopLayer(popover.value)).catch(() => undefined)
  })

  onBeforeUnmount(() => win.removeEventListener('resize', place))

  function place(): void {
    const rect = measureRect(trigger.value)
    if (!rect) return
    style.value = {
      top: `calc(${rect.bottom}px + var(--popover-offset))`,
      right: `${measureWindow().width - rect.right}px`,
    }
  }

  return { style }
}
