/** The listeners a node's parts share, so the note in its place and the note in the layer over the others answer alike. */
import { mergeListeners } from './destination'
import type { useHoverFocus } from './hoverFocus'
import type { useNodePress } from './press'

export function createNodeEvents(
  press: ReturnType<typeof useNodePress>,
  hoverFocus: ReturnType<typeof useHoverFocus>,
) {
  const shared = {
    click: press.handlers.onClick,
    dblclick: press.handlers.onDoubleClick,
    contextmenu: press.handlers.onContextMenu,
    pointerenter: hoverFocus.onPointerEnter,
    pointerleave: hoverFocus.onPointerLeave,
    focusin: hoverFocus.onFocusIn,
    focusout: hoverFocus.onFocusOut,
  }
  return mergeListeners(
    shared as unknown as Record<string, (event: PointerEvent) => void>,
    press.listening,
  )
}
