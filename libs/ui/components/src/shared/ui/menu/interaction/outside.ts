/**
 * A press that lands outside an element, reported while the element is active.
 *
 * The window is listened to from mount to unmount; a press with no element
 * drawn yet is outside.
 */
import { onBeforeUnmount, onMounted } from 'vue'
import type { ShallowRef } from 'vue'

/**
 * Whether an event began inside the element. A press inside an open shadow root reaches the window
 * with the shadow host as its target, so the path of the event is read as well.
 */
export function isInside(element: HTMLElement | null, event: Event): boolean {
  if (!element) return false
  if (event.composedPath().includes(element)) return true
  const target = event.target
  return target instanceof Node && element.contains(target)
}

export function useOutsidePress(
  element: Readonly<ShallowRef<HTMLElement | null>>,
  onOutside: () => void,
  isActive: () => boolean = () => true,
): void {
  const onPress = (event: Event): void => {
    if (!isActive() || isInside(element.value, event)) return
    onOutside()
  }

  onMounted(() => window.addEventListener('pointerdown', onPress, true))
  onBeforeUnmount(() => window.removeEventListener('pointerdown', onPress, true))
}
