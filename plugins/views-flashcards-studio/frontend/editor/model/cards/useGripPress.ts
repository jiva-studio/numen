import { onScopeDispose, shallowRef, type Ref } from 'vue'
import { win } from '@numen/ui'

/** What a press lands on that is worked, and lets the press have it. */
const WORKED = 'input, textarea, button, a, select, [contenteditable]'

export interface GripPress {
  readonly isDraggable: Readonly<Ref<boolean>>
  readonly press: (event: PointerEvent) => void
}

/** Lets a press on something worked inside a draggable strip have the press: the strip is not draggable from that press until the pointer is released. */
export function useGripPress(): GripPress {
  const isDraggable = shallowRef(true)

  const release = (): void => {
    isDraggable.value = true
    win.removeEventListener('pointerup', release)
    win.removeEventListener('pointercancel', release)
  }

  onScopeDispose(release)

  const press = (event: PointerEvent): void => {
    if ((event.target as HTMLElement | null)?.closest?.(WORKED) == null) return
    isDraggable.value = false
    win.addEventListener('pointerup', release)
    win.addEventListener('pointercancel', release)
  }

  return { isDraggable, press }
}
