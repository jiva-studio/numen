/**
 * A tab lifted off its strip, where letting go of it would land it, and what
 * the workspace becomes once it is let go.
 *
 * Where the pointer is over is asked of the document, in the coordinates of the
 * frame the workspace is drawn in.
 */
import { computed, type ComputedRef, type Ref, type ShallowRef } from 'vue'
import type { Clock } from '@/shared/lib/clock'
import { usePressDrag } from '@/shared/ui/drag-preview'
import type { Position } from '@/shared/lib/geometry'
import {
  getCaretRect,
  findEdgeSide,
  getOverlay,
  getRect,
  getSide,
  getSlot,
  type TabDropTarget,
} from '../lib/drop'
import { dropOnEdge, dropTab, moveTabWithin, type NodeIdFactory } from '../lib/edit'
import type { NodeId, Tab, TabId, Workspace } from '../lib/node'
import type { Rect } from '../lib/rect'

/** A tab under the pointer, and the pane its strip belongs to. */
interface Drag {
  readonly tab: TabId
  readonly from: NodeId
}

export interface TabDragOptions {
  /** The workspace the drag reads and writes. */
  readonly workspace: Ref<Workspace>
  /** The element the whole workspace is drawn in. */
  readonly frame: Readonly<ShallowRef<HTMLElement | null>>
  /** What a tab is, for the name drawn at the pointer. */
  readonly getTab: (id: TabId) => Tab | undefined
  /** Where identities for what a gesture makes come from. */
  readonly getIdFactory: () => NodeIdFactory
  /** How close to the outer edge divides the whole workspace. */
  readonly getEdge: () => number
  /** How far the pointer travels before a press becomes a drag. */
  readonly getThreshold: () => number
  readonly getClock: () => Clock
  /** Called when a tab is dragged and released outside of any workspace drop target. */
  readonly onTearOff?: ((tab: TabId, at: Position) => void) | undefined
}

export interface TabDragState {
  /** Whether the press being made has travelled far enough to be a drag. */
  readonly hasMoved: ComputedRef<boolean>
  /** The area a drop would take, in the coordinates of the frame. */
  readonly overlay: ComputedRef<Rect | null>
  /** The name drawn at the pointer, and nothing until a press has become a drag. */
  readonly label: ComputedRef<string | null>
  /** Where the pointer is. */
  readonly position: Readonly<Ref<Position | null>>
  /** Where the tab would land. */
  readonly dropTarget: Readonly<Ref<TabDropTarget | null>>
  /** A press on a tab, which a move turns into a drag. */
  readonly press: (tab: TabId, at: PointerEvent) => void
}

export function useTabDrag(options: TabDragOptions): TabDragState {
  let lastPosition: Position = { x: 0, y: 0 }

  const {
    dragging,
    at: dropTarget,
    position,
    lift,
  } = usePressDrag<Drag, TabDropTarget>({
    getThreshold: options.getThreshold,
    getClock: options.getClock,
    getDropTargetAt: (_item, at) => {
      lastPosition = at
      return resolveDropTarget(at, options.frame.value, options.getEdge)
    },
    commit: (item, at) => {
      if (at) {
        land(item, at)
      } else if (options.onTearOff) {
        const screenX = typeof window !== 'undefined' ? window.screenX : 0
        const screenY = typeof window !== 'undefined' ? window.screenY : 0
        options.onTearOff(item.tab, {
          x: screenX + lastPosition.x,
          y: screenY + lastPosition.y,
        })
      }
    },
  })

  const hasMoved = computed(() => dragging.value?.hasMoved === true)
  const overlay = computed(() => (hasMoved.value ? (dropTarget.value?.box ?? null) : null))
  const label = computed(() => {
    const held = dragging.value
    if (!held?.hasMoved) return null
    return options.getTab(held.item.tab)?.title ?? held.item.tab
  })

  function press(tab: TabId, at: PointerEvent): void {
    if (at.button !== 0) return

    const holder = document.elementFromPoint
      ? document.elementFromPoint(at.clientX, at.clientY)?.closest('[data-workspace-pane]')
      : null
    const from = holder?.getAttribute('data-workspace-pane') ?? options.workspace.value.focus

    lift({ tab, from }, at)
  }

  function land(drag: Drag, at: TabDropTarget): void {
    options.workspace.value = landTab(options.workspace.value, {
      drag,
      at,
      ids: options.getIdFactory(),
    })
  }

  return { hasMoved, overlay, label, position, dropTarget, press }
}

/** A box put in the coordinates of the frame. */
type ToLocal = (box: Rect) => Rect

interface StripTarget {
  readonly x: number
  readonly strip: Element
  readonly pane: NodeId
}

const isInside = (at: Position, box: Rect): boolean =>
  at.x >= box.x && at.x <= box.x + box.width && at.y >= box.y && at.y <= box.y + box.height

const getClosest = (under: Element | null, selector: string): Element | null =>
  under?.closest(selector) ?? null

/** A tab put in order among the tabs already drawn on a strip. */
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

/** A pane divided along the side the pointer is nearest. */
function getPaneDropTarget(
  at: Position,
  pane: { readonly element: Element; readonly id: NodeId },
  local: ToLocal,
): TabDropTarget {
  const box = getRect(pane.element)
  const side = getSide(at, box)
  return { kind: 'pane', pane: pane.id, side, box: local(getOverlay(side, box)) }
}

interface LandAction {
  readonly drag: Drag
  readonly at: TabDropTarget
  readonly ids: NodeIdFactory
}

function landTab(workspace: Workspace, action: LandAction): Workspace {
  const { drag, at, ids } = action
  if (at.kind === 'edge') {
    return dropOnEdge(workspace, { tab: drag.tab, side: at.side }, ids)
  }

  if (at.kind === 'pane') {
    return dropTab(workspace, { tab: drag.tab, onto: at.pane, side: at.side }, ids)
  }

  const joined =
    at.pane === drag.from
      ? workspace
      : dropTab(workspace, { tab: drag.tab, onto: at.pane, side: 'center' }, ids)
  return moveTabWithin(joined, drag.tab, at.slot)
}

/**
 * What the pointer is over, asked of the document.
 *
 * A strip is read first, so that a tab can be put in order among its
 * neighbours; then the outer edge, which divides the whole workspace; then the
 * pane, which divides itself.
 */
function resolveDropTarget(
  at: Position,
  frame: Element | null,
  getEdge: () => number,
): TabDropTarget | null {
  if (!frame) return null

  const outer = getRect(frame)
  if (!isInside(at, outer)) return null

  const local = (box: Rect): Rect => ({ ...box, x: box.x - outer.x, y: box.y - outer.y })
  const elemAtPoint = document.elementFromPoint ? document.elementFromPoint(at.x, at.y) : null

  const strip = getClosest(elemAtPoint, '[data-workspace-strip]')
  const pane = getClosest(elemAtPoint, '[data-workspace-pane]')
  const id = pane?.getAttribute('data-workspace-pane')

  if (strip && id) return getStripDropTarget({ x: at.x, strip, pane: id }, local)

  const side = findEdgeSide(at, outer, getEdge())
  if (side) return { kind: 'edge', side, box: local(getOverlay(side, outer)) }

  if (!pane || !id) return null

  return getPaneDropTarget(at, { element: pane, id }, local)
}
