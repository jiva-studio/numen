import { computed, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import type { Clock } from '@numen/ui'
import { usePressDrag } from '@numen/ui'
import type { Position, Size } from '@numen/ui'
import {
  getCaretRect,
  findEdgeSide,
  getOverlay,
  getSide,
  getSlot,
  type TabDropTarget,
} from '../lib/drop'
import { getRect } from '../api/rect'
import {
  dropOnEdge,
  dropTab as dropTabInTree,
  moveTabWithin,
  type NodeIdFactory,
} from '../lib/edit'
import type { NodeId, Tab, TabId, Workspace } from '../lib/node'
import type { Rect } from '../lib/rect'
import { getViewportSize, getWindowOrigin } from '@/shared/api/dom'

interface Drag {
  readonly tab: TabId
  readonly from: NodeId
}

/** Options for workspace tab drag state. */
export interface TabDragOptions {
  readonly workspace: Ref<Workspace>
  readonly frame: Readonly<ShallowRef<HTMLElement | null>>
  readonly getTab: (id: TabId) => Tab | undefined
  readonly getIdFactory: () => NodeIdFactory
  readonly getEdge: () => number
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  readonly getViewport?: () => Size
  readonly onTearOff?: ((tab: TabId, pos: Position) => void) | undefined
}

/** Reactive state and handlers for dragging a workspace tab. */
export interface TabDragState {
  readonly hasMoved: ComputedRef<boolean>
  readonly overlay: ComputedRef<Rect | null>
  readonly label: ComputedRef<string | null>
  readonly position: Readonly<Ref<Position | null>>
  readonly dropTarget: Readonly<Ref<TabDropTarget | null>>
  readonly press: (tab: TabId, pointer: PointerEvent) => void
}

/** Tracks tab drag gestures and updates the workspace layout on drop. */
export function useTabDrag(options: TabDragOptions): TabDragState {
  const getViewport = options.getViewport ?? getViewportSize

  const {
    dragging,
    at: dropTarget,
    position,
    startPress,
  } = usePressDrag<Drag, TabDropTarget>({
    getThreshold: options.getThreshold,
    getClock: options.getClock,
    getDropTargetAt: (_item, point) =>
      isInsideViewport(point, getViewport())
        ? resolveDropTarget(point, options.frame.value, options.getEdge)
        : null,
    commit: (item, target, point) => {
      if (target) {
        dropTab(item, target)
      } else if (options.onTearOff && !isInsideViewport(point, getViewport())) {
        options.onTearOff(item.tab, getScreenPosition(point))
      }
    },
  })

  const hasMoved = computed(() => dragging.value?.hasMoved === true)
  const overlay = computed(() => (hasMoved.value ? (dropTarget.value?.box ?? null) : null))
  const label = computed(() => {
    const target = dragging.value
    if (!target?.hasMoved) return null
    return options.getTab(target.item.tab)?.title ?? target.item.tab
  })

  function press(tab: TabId, pointer: PointerEvent): void {
    if (pointer.button !== 0) return

    const holder = document
      .elementFromPoint(pointer.clientX, pointer.clientY)
      ?.closest('[data-workspace-pane]')
    const from = holder?.getAttribute('data-workspace-pane') ?? options.workspace.value.focus

    startPress({ tab, from }, pointer)
  }

  function dropTab(drag: Drag, target: TabDropTarget): void {
    options.workspace.value = applyTabDrop(options.workspace.value, {
      drag,
      target,
      ids: options.getIdFactory(),
    })
  }

  return { hasMoved, overlay, label, position, dropTarget, press }
}

type ToLocal = (box: Rect) => Rect

interface StripTarget {
  readonly x: number
  readonly strip: Element
  readonly pane: NodeId
}

const isInside = (point: Position, box: Rect): boolean =>
  point.x >= box.x &&
  point.x <= box.x + box.width &&
  point.y >= box.y &&
  point.y <= box.y + box.height

const isInsideViewport = (point: Position, viewport: Size): boolean =>
  point.x >= 0 && point.y >= 0 && point.x < viewport.width && point.y < viewport.height

const getClosestElement = (targetElement: Element | null, selector: string): Element | null =>
  targetElement?.closest(selector) ?? null

function getStripDropTarget(target: StripTarget, local: ToLocal): TabDropTarget {
  const { x, strip, pane } = target
  const tabs = [...strip.children].map((tab) => getRect(tab))
  const slot = getSlot(x, tabs)
  return {
    kind: 'strip',
    pane,
    slot,
    box: local(getCaretRect(slot, tabs, getRect(strip))),
  }
}

function getPaneDropTarget(
  point: Position,
  pane: { readonly element: Element; readonly id: NodeId },
  local: ToLocal,
): TabDropTarget {
  const box = getRect(pane.element)
  const side = getSide(point, box)
  return { kind: 'pane', pane: pane.id, side, box: local(getOverlay(side, box)) }
}

interface DropAction {
  readonly drag: Drag
  readonly target: TabDropTarget
  readonly ids: NodeIdFactory
}

function applyTabDrop(workspace: Workspace, action: DropAction): Workspace {
  const { drag, target, ids } = action
  if (target.kind === 'edge') {
    return dropOnEdge(workspace, { tab: drag.tab, side: target.side }, ids)
  }

  if (target.kind === 'pane') {
    return dropTabInTree(workspace, { tab: drag.tab, onto: target.pane, side: target.side }, ids)
  }

  const joined =
    target.pane === drag.from
      ? workspace
      : dropTabInTree(workspace, { tab: drag.tab, onto: target.pane, side: 'center' }, ids)
  return moveTabWithin(joined, drag.tab, target.slot)
}

function resolveDropTarget(
  point: Position,
  frame: Element | null,
  getEdge: () => number,
): TabDropTarget | null {
  if (!frame) return null

  const outer = getRect(frame)
  if (!isInside(point, outer)) return null

  const local = (box: Rect): Rect => ({ ...box, x: box.x - outer.x, y: box.y - outer.y })
  const underElement = document.elementFromPoint(point.x, point.y)

  const strip = getClosestElement(underElement, '[data-workspace-strip]')
  const pane = getClosestElement(underElement, '[data-workspace-pane]')
  const id = pane?.getAttribute('data-workspace-pane')

  if (strip && id) return getStripDropTarget({ x: point.x, strip, pane: id }, local)

  const side = findEdgeSide(point, outer, getEdge())
  if (side) return { kind: 'edge', side, box: local(getOverlay(side, outer)) }

  if (!pane || !id) return null

  return getPaneDropTarget(point, { element: pane, id }, local)
}

const getScreenPosition = (point: Position): Position => {
  const origin = getWindowOrigin()
  return { x: origin.x + point.x, y: origin.y + point.y }
}
