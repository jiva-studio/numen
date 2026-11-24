/**
 * The position of the popover of an info button over the page: fixed, below the button and flipped
 * above it where there is no room below, kept inside the window.
 */
import { ref, type Ref } from 'vue'
import { measureRect, measureWindow } from '@numen/ui'

const GAP = 4
const MARGIN = 8

export interface InfoPopoverPlace {
  readonly style: Ref<Record<string, string>>
  readonly place: (button: Element | null, popover: Element | null) => void
}

export function useInfoPopoverPlace(): InfoPopoverPlace {
  const style = ref<Record<string, string>>({ left: '0px', top: '0px' })

  function place(button: Element | null, popover: Element | null): void {
    const anchor = measureRect(button)
    if (!anchor) return
    const size = measureRect(popover) ?? { width: 0, height: 0 }
    const room = measureWindow()
    const below = anchor.bottom + GAP
    const above = anchor.top - GAP - size.height
    const isFlipped = below + size.height > room.height - MARGIN && above >= MARGIN
    const top = isFlipped ? above : below
    const left = Math.min(anchor.left, room.width - MARGIN - size.width)
    style.value = {
      left: `${Math.max(left, MARGIN)}px`,
      top: `${Math.max(top, MARGIN)}px`,
    }
  }

  return { style, place }
}
