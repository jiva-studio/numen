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

  function onPointerDown(event: PointerEvent) {
    const target = event.target
    if (!(target instanceof Element)) return
    const nodeId = target.closest('[data-node]')?.getAttribute('data-node')
    if (!nodeId) return
    const path = state.canvas.widgets.getPath(nodeId)
    if (path) state.canvas.view.warm?.(path)
  }

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
    if (!state.canvas.empty.value) return
    event.preventDefault()
    state.menu.openMenu({
      node: null,
      at: { x: event.clientX, y: event.clientY },
      opening: 'pointer',
    })
  }

  function onEdgeMenu(pair: string, position: { x: number; y: number }) {
    const rect = measureRect(rootElement.value)
    const anchor = rect ? { x: position.x - rect.left, y: position.y - rect.top } : position
    state.inspector.openLinkInspector(pair, anchor).catch(state.reportFailure)
  }

  function closeMenu(id?: string) {
    const node = state.menu.request.value?.node ?? null
    if (id === undefined) state.menu.dismiss()
    else state.menu.chooseMenuItem(id)
    if (node !== null) picture.value?.focusNode(node)
  }

  return {
    rootElement,
    picture,
    lastPointer,
    onPointerDown,
    onPointerUp,
    onContextMenu,
    onEdgeMenu,
    closeMenu,
  }
}
