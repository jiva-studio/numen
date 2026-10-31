/**
 * Canvas interaction and viewport gestures for the plex graph view.
 */
import { measureRect } from '@numen/ui'
import { ref, useTemplateRef } from 'vue'
import type { PlexTabState } from '../types'

export interface PlexCanvasOptions {
  readonly state: PlexTabState
}

export function usePlexCanvas(options: PlexCanvasOptions) {
  const { state } = options
  const rootElement = useTemplateRef<HTMLElement>('rootElement')
  const picture = useTemplateRef<{ focusNode: (id: string) => void }>('picture')
  const lastPointer = ref<{ x: number; y: number }>({ x: 300, y: 300 })

  function onPointerUp(event: PointerEvent) {
    const rect = measureRect(rootElement.value)
    if (rect) {
      lastPointer.value = {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      }
    } else {
      lastPointer.value = { x: event.clientX, y: event.clientY }
    }
  }

  function onContextMenu(event: MouseEvent) {
    if (!state.empty.value) return
    event.preventDefault()
    state.openMenu({
      node: null,
      at: { x: event.clientX, y: event.clientY },
      opening: 'pointer',
    })
  }

  function onEdgeMenu(pair: string, position: { x: number; y: number }) {
    const rect = measureRect(rootElement.value)
    if (rect) {
      const localX = position.x - rect.left
      const localY = position.y - rect.top
      state.openLinkInspector(pair, { x: localX, y: localY })
    } else {
      state.openLinkInspector(pair, position)
    }
  }

  function closeMenu(id?: string) {
    const node = state.menu.value?.node ?? null
    if (id === undefined) state.dismiss()
    else state.chooseMenuItem(id)
    if (node !== null) picture.value?.focusNode(node)
  }

  return {
    rootElement,
    picture,
    lastPointer,
    onPointerUp,
    onContextMenu,
    onEdgeMenu,
    closeMenu,
  }
}
