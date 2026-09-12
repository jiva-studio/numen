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
  caretAt,
  edgeOf,
  overlayFor,
  rectOf,
  sideAt,
  slotAt,
  type TabLanding,
} from '../lib/drop'
import {
  dropOnEdge,
  dropTab,
  moveTabWithin,
  type NodeIdFactory,
} from '../lib/edit'
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
  readonly tabOf: (id: TabId) => Tab | undefined
  /** Where identities for what a gesture makes come from. */
  readonly naming: () => NodeIdFactory
  /** How close to the outer edge divides the whole workspace. */
  readonly edge: () => number
  /** How far the pointer travels before a press becomes a drag. */
  readonly threshold: () => number
  readonly clock: () => Clock
}

export interface TabDragState {
  /** Whether the press being made has travelled far enough to be a drag. */
  readonly moved: ComputedRef<boolean>
  /** The area a drop would take, in the coordinates of the frame. */
  readonly overlay: ComputedRef<Rect | null>
  /** The name drawn at the pointer, and nothing until a press has become a drag. */
  readonly label: ComputedRef<string | null>
  /** Where the pointer is. */
  readonly position: Readonly<Ref<Position | null>>
  /** Where the tab would land. */
  readonly landing: Readonly<Ref<TabLanding | null>>
  /** A press on a tab, which a move turns into a drag. */
  readonly press: (tab: TabId, at: PointerEvent) => void
}

export function useTabDrag(options: TabDragOptions): TabDragState {
  const { dragging, at: landing, position, lift } = usePressDrag<Drag, TabLanding>({
    threshold: options.threshold,
    clock: options.clock,
    landingAt: (_item, at) => landingAt(at.x, at.y),
    settle: (item, at) => {
      if (at) land(item, at)
    },
  })

  const moved = computed(() => dragging.value?.moved === true)

  const overlay = computed(() => (moved.value ? (landing.value?.box ?? null) : null))

  const label = computed(() => {
    const held = dragging.value
    if (!held?.moved) return null
    return options.tabOf(held.item.tab)?.title ?? held.item.tab
  })

  function press(tab: TabId, at: PointerEvent): void {
    if (at.button !== 0) return

    const holder = document.elementFromPoint(at.clientX, at.clientY)?.closest('[data-workspace-pane]')
    const from = holder?.getAttribute('data-workspace-pane') ?? options.workspace.value.focus

    lift({ tab, from }, at)
  }

  function land(drag: Drag, at: TabLanding): void {
    const ids = options.naming()
    const workspace = options.workspace

    if (at.kind === 'edge') {
      workspace.value = dropOnEdge(workspace.value, drag.tab, at.side, ids)
      return
    }

    if (at.kind === 'pane') {
      workspace.value = dropTab(workspace.value, { tab: drag.tab, onto: at.pane, side: at.side }, ids)
      return
    }

    const joined =
      at.pane === drag.from
        ? workspace.value
        : dropTab(workspace.value, { tab: drag.tab, onto: at.pane, side: 'center' }, ids)
    workspace.value = moveTabWithin(joined, drag.tab, at.slot)
  }

  /**
   * What the pointer is over, asked of the document.
   *
   * A strip is read first, so that a tab can be put in order among its
   * neighbours; then the outer edge, which divides the whole workspace; then the
   * pane, which divides itself.
   */
  function landingAt(x: number, y: number): TabLanding | null {
    const held = options.frame.value
    if (!held) return null

    const outer = rectOf(held)
    const inside =
      x >= outer.x && x <= outer.x + outer.width && y >= outer.y && y <= outer.y + outer.height
    if (!inside) return null

    const local = (box: Rect): Rect => ({ ...box, x: box.x - outer.x, y: box.y - outer.y })
    const under = document.elementFromPoint(x, y)

    const strip = under?.closest('[data-workspace-strip]')
    const pane = under?.closest('[data-workspace-pane]')
    const id = pane?.getAttribute('data-workspace-pane')

    if (strip && id) {
      // A strip holds tabs and nothing else, in the order they are drawn.
      const tabs = [...strip.children].map((tab) => rectOf(tab))
      const slot = slotAt(x, tabs)
      return { kind: 'strip', pane: id, slot, box: local(caretAt(slot, tabs, rectOf(strip))) }
    }

    const side = edgeOf({ x, y }, rectOf(held), options.edge())
    if (side) return { kind: 'edge', side, box: local(overlayFor(side, rectOf(held))) }

    if (!pane || !id) return null

    const box = rectOf(pane)
    const asked = sideAt({ x, y }, box)
    return { kind: 'pane', pane: id, side: asked, box: local(overlayFor(asked, box)) }
  }

  return { moved, overlay, label, position, landing, press }
}
