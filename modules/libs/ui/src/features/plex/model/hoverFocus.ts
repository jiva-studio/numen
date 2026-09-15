/**
 * Whether an element is hovered or holds visible keyboard focus, which the
 * plex draws the same way. Focus stays while it moves to something inside.
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

export function useHoverFocus() {
  const isHovered = ref(false)
  const isKeyboardFocused = ref(false)

  return {
    /** The hand is over it, or the keyboard is visibly on it. */
    isOn: computed(() => isHovered.value || isKeyboardFocused.value),

    onPointerEnter: () => {
      isHovered.value = true
    },

    onPointerLeave: () => {
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
