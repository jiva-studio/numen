/**
 * Whether an element is hovered or holds visible keyboard focus, which the
 * plex draws the same way. Focus stays while it moves to something inside.
 *
 * `parts` are the elements the node is drawn in: a pointer that leaves one onto another is still over the node.
 */
import { computed, ref } from 'vue'

function isKeyboardOn(element: Element): boolean {
  try {
    return element.matches(':focus-visible')
  } catch {
    // Unsupported selectors mean no visible keyboard focus is reported.
    return false
  }
}

export function useHoverFocus(parts: () => readonly (Element | null)[] = () => []) {
  const isHovered = ref(false)
  const isKeyboardFocused = ref(false)

  return {
    /** The pointer is over it, or keyboard focus is visibly on it. */
    isOn: computed(() => isHovered.value || isKeyboardFocused.value),

    /** Takes it off, whatever pointer or focus events were or were not delivered. */
    release: () => {
      isHovered.value = false
      isKeyboardFocused.value = false
    },

    onPointerEnter: () => {
      isHovered.value = true
    },

    onPointerLeave: (event?: PointerEvent) => {
      const next = event?.relatedTarget
      if (next instanceof Node && parts().some((part) => part?.contains(next))) return
      isHovered.value = false
    },

    onFocusIn: (event: FocusEvent) => {
      isKeyboardFocused.value = isKeyboardOn(event.target as Element)
    },

    // The keyboard stays on the element while it moves to something inside it.
    onFocusOut: (event: FocusEvent) => {
      const within = event.currentTarget as Element
      const next = event.relatedTarget as Node | null
      isKeyboardFocused.value = !!(next && within.contains(next))
    },
  }
}
