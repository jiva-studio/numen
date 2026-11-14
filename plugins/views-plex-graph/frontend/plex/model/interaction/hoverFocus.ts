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

  const isInside = (within: EventTarget | null, next: EventTarget | null): boolean => {
    if (!(next instanceof Node)) return false
    if (within instanceof Node && within.contains(next)) return true
    return parts().some((part) => part?.contains(next))
  }

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

    // Focus a script moves inside the node, as useLift does, need not match :focus-visible.
    onFocusIn: (event: FocusEvent) => {
      if (isKeyboardFocused.value && isInside(event.currentTarget, event.relatedTarget)) return
      isKeyboardFocused.value = isKeyboardOn(event.target as Element)
    },

    // The lifted part is teleported out of the group, so moving into it is moving inside too.
    onFocusOut: (event: FocusEvent) => {
      isKeyboardFocused.value = isInside(event.currentTarget, event.relatedTarget)
    },
  }
}
