/**
 * The Cmd/Ctrl+drag of the entries of the slots. A press with the modifier on an entry the plugins
 * mark starts a press; past the threshold it is a drag, and a drop on a marked slot writes the lists
 * the move changes in one write. Every other press is left to the page.
 */
import { computed, onScopeDispose, shallowRef, type ComputedRef, type Ref } from 'vue'
import { browserClock, measureRect, usePressDrag, type Clock, type Position } from '@numen/ui'
import {
  SLOT,
  SLOT_ATTRIBUTE,
  SLOT_ENTRY_ATTRIBUTE,
  moveSlotEntry,
  type SlotName,
  type SlotLayout,
  type SlotLayoutChange,
} from '@numen/plugin-host'
import { findDropPlace, type EntryBox } from '../lib/dropPlace'

/** The attribute the root element of the page carries while a slot entry is dragged. */
export const DRAGGING_ATTRIBUTE = 'data-numen-slot-dragging'

/** How far, in pixels, the pointer moves before a press becomes a drag. */
const DRAG_THRESHOLD = 4

export interface SlotDragDeps {
  readonly isMac: () => boolean
  readonly findLayoutKey: (itemKey: string, slot: SlotName) => string | undefined
  readonly listLayoutKeys: (slot: SlotName) => readonly string[]
  readonly getLayout: () => SlotLayout
  readonly writeLayout: (change: SlotLayoutChange) => Promise<void>
  readonly getClock?: () => Clock
}

/** Where the insertion marker of a drop is drawn, in the coordinates of the viewport. */
export interface SlotDropMarker {
  readonly x: number
  readonly top: number
  readonly height: number
}

/** What the layer of the drag draws: the dragged entry at the pointer and the marker of the place it goes to. */
export interface SlotDragState {
  readonly source: ComputedRef<Element | null>
  readonly position: Readonly<Ref<Position | null>>
  readonly marker: ComputedRef<SlotDropMarker | null>
}

/** The entry a press holds: its layout key, the slot it is drawn in and its element. */
interface PressedEntry {
  readonly key: string
  readonly from: SlotName
  readonly element: Element
}

interface SlotDropTarget {
  readonly slot: SlotName
  readonly before: string | undefined
  readonly marker: SlotDropMarker
}

const SLOT_NAMES: readonly string[] = Object.values(SLOT)

const findSlot = (element: Element | null): { element: Element; slot: SlotName } | null => {
  const slotElement = element?.closest(`[${SLOT_ATTRIBUTE}]`)
  const slot = slotElement?.getAttribute(SLOT_ATTRIBUTE)
  if (!slotElement || !slot || !SLOT_NAMES.includes(slot)) return null
  return { element: slotElement, slot: slot as SlotName }
}

const hasDragModifier = (event: PointerEvent, isMac: boolean): boolean =>
  isMac ? event.metaKey : event.ctrlKey

type LayoutDeps = Pick<
  SlotDragDeps,
  'findLayoutKey' | 'listLayoutKeys' | 'getLayout' | 'writeLayout'
>

const findPressedEntry = (target: EventTarget | null, deps: LayoutDeps): PressedEntry | null => {
  if (!(target instanceof Element)) return null
  const entry = target.closest(`[${SLOT_ENTRY_ATTRIBUTE}]`)
  const found = findSlot(entry)
  if (!entry || !found) return null
  const key = deps.findLayoutKey(entry.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '', found.slot)
  if (key === undefined) return null
  return { key, from: found.slot, element: entry }
}

const listEntryBoxes = (
  slotElement: Element,
  slot: SlotName,
  dragged: string,
  deps: LayoutDeps,
): EntryBox[] =>
  [...slotElement.querySelectorAll(`[${SLOT_ENTRY_ATTRIBUTE}]`)].flatMap((element) => {
    const key = deps.findLayoutKey(element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '', slot)
    if (key === undefined || key === dragged) return []
    const box = measureRect(element)
    return [{ key, left: box.left, right: box.right }]
  })

/** The slot under the point and the place in it the entry goes to; null outside the marked slots. */
const findDropTarget = (
  entry: PressedEntry,
  point: Position,
  deps: LayoutDeps,
): SlotDropTarget | null => {
  const found = findSlot(document.elementFromPoint(point.x, point.y))
  if (!found) return null
  const box = measureRect(found.element)
  const place = findDropPlace(
    listEntryBoxes(found.element, found.slot, entry.key, deps),
    point.x,
    box.left,
  )
  return {
    slot: found.slot,
    before: place.before,
    marker: { x: place.x, top: box.top, height: box.height },
  }
}

/** Writes the lists a drop changes, in one write; a drop that changes nothing writes nothing. */
const writeDrop = (entry: PressedEntry, target: SlotDropTarget, deps: LayoutDeps): void => {
  const drawnKeys: Partial<Record<SlotName, readonly string[]>> = {
    [entry.from]: deps.listLayoutKeys(entry.from),
    [target.slot]: deps.listLayoutKeys(target.slot),
  }
  const change = moveSlotEntry(deps.getLayout(), drawnKeys, {
    key: entry.key,
    from: entry.from,
    to: target.slot,
    before: target.before,
  })
  if (Object.keys(change).length === 0) return
  // The settings store reports a failed write itself and reads the file again.
  deps.writeLayout(change).catch(() => undefined)
}

/** Stops the click the browser sends after the release that ends a drag, so no entry runs it. */
function createClickBlocker() {
  let isBlocked = false
  return {
    block: (): void => {
      isBlocked = true
      globalThis.setTimeout(() => {
        isBlocked = false
      }, 0)
    },
    onClick: (event: MouseEvent): void => {
      if (!isBlocked) return
      isBlocked = false
      event.preventDefault()
      event.stopImmediatePropagation()
    },
  }
}

interface DocumentListeners {
  readonly blur: () => void
  readonly pointerdown: (event: PointerEvent) => void
  readonly keydown: (event: KeyboardEvent) => void
  readonly click: (event: MouseEvent) => void
}

/** Adds the listeners to the document in the capture phase, before any element of the page, until disposal. */
const listenInCapture = (listeners: DocumentListeners): void => {
  window.addEventListener('blur', listeners.blur)
  document.addEventListener('pointerdown', listeners.pointerdown, true)
  document.addEventListener('keydown', listeners.keydown, true)
  document.addEventListener('click', listeners.click, true)
  onScopeDispose(() => {
    window.removeEventListener('blur', listeners.blur)
    document.removeEventListener('pointerdown', listeners.pointerdown, true)
    document.removeEventListener('keydown', listeners.keydown, true)
    document.removeEventListener('click', listeners.click, true)
  })
}

export function createSlotDrag(deps: SlotDragDeps): SlotDragState {
  const root = document.documentElement
  const isCancelled = shallowRef(false)
  const clicks = createClickBlocker()
  const endDrag = (): void => root.removeAttribute(DRAGGING_ATTRIBUTE)
  const cancelDrag = (): void => {
    isCancelled.value = true
    endDrag()
  }

  const {
    dragging,
    at: dropTarget,
    position,
    startPress,
    cancelPress,
  } = usePressDrag<PressedEntry, SlotDropTarget>({
    getThreshold: () => DRAG_THRESHOLD,
    getClock: () => deps.getClock?.() ?? browserClock,
    getDropTargetAt: (entry, point) =>
      isCancelled.value ? null : findDropTarget(entry, point, deps),
    begin: () => root.setAttribute(DRAGGING_ATTRIBUTE, ''),
    commit: (entry, target) => {
      endDrag()
      clicks.block()
      if (target && !isCancelled.value) writeDrop(entry, target, deps)
    },
    cancel: endDrag,
  })

  const abandonDrag = (): void => {
    cancelDrag()
    cancelPress()
  }

  listenInCapture({
    // The window loses the pointer, and its release, when it loses focus.
    blur: abandonDrag,
    pointerdown: (event: PointerEvent): void => {
      // A release that never arrived leaves a drag open; a new press ends it.
      if (root.hasAttribute(DRAGGING_ATTRIBUTE)) abandonDrag()
      if (event.button !== 0 || !hasDragModifier(event, deps.isMac())) return
      const entry = findPressedEntry(event.target, deps)
      if (!entry) return
      event.preventDefault()
      event.stopPropagation()
      isCancelled.value = false
      startPress(entry, event)
    },
    keydown: (event: KeyboardEvent): void => {
      if (event.key !== 'Escape' || !dragging.value?.hasMoved || isCancelled.value) return
      event.preventDefault()
      event.stopPropagation()
      cancelDrag()
    },
    click: clicks.onClick,
  })
  onScopeDispose(endDrag)

  const isDrawn = (): boolean => dragging.value?.hasMoved === true && !isCancelled.value
  return {
    source: computed(() => (isDrawn() ? (dragging.value?.item.element ?? null) : null)),
    position,
    marker: computed(() => (isDrawn() ? (dropTarget.value?.marker ?? null) : null)),
  }
}
