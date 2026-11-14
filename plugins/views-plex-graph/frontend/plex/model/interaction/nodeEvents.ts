/** The pointer listeners a node's parts share, so the note in its place and the note in the layer over the others answer alike. */
import { mergeListeners } from './destination'
import type { useHoverFocus } from './hoverFocus'
import type { useNodePress } from './press'

type Listener = (event: PointerEvent) => void

const on = (handler: (event: never) => void) => handler as Listener

export function getNodeEvents(
  press: ReturnType<typeof useNodePress>,
  hoverFocus: ReturnType<typeof useHoverFocus>,
) {
  return mergeListeners(
    {
      click: on(press.handlers.onClick),
      dblclick: on(press.handlers.onDoubleClick),
      contextmenu: on(press.handlers.onContextMenu),
      pointerenter: on(hoverFocus.onPointerEnter),
      pointerleave: on(hoverFocus.onPointerLeave),
    },
    press.listening,
  )
}
