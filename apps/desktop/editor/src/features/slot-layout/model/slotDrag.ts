/**
 * The Cmd/Ctrl+drag of the entries of the slots. A press with the modifier on an entry the plugins
 * mark starts a press; past the threshold it is a drag, and a drop on a marked slot writes the lists
 * the move changes in one write. While the drag goes on, the entry is drawn where a drop would put it.
 * Every other press is left to the page.
 */
import { onScopeDispose, shallowRef, watch } from 'vue'
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
  /** Draws the slots with the lists of the change over the layout; null draws the layout as it is. */
  readonly showPreview?: (change: SlotLayoutChange | null) => void
  readonly getClock?: () => Clock
}

/** The entry a press holds: its layout key and the slot it is drawn in. */
interface PressedEntry {
  readonly key: string
  readonly from: SlotName
}

interface SlotDropTarget {
  readonly slot: SlotName
  readonly before: string | undefined
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
  return { key, from: found.slot }
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
  return { slot: found.slot, before: place.before }
}

/** The lists that moving the entry, drawn in `from`, to the target changes; none when it stays. */
const getMoveChange = (
  entry: PressedEntry,
  from: SlotName,
  target: SlotDropTarget,
  deps: LayoutDeps,
): SlotLayoutChange => {
  const drawnKeys: Partial<Record<SlotName, readonly string[]>> = {
    [from]: deps.listLayoutKeys(from),
    [target.slot]: deps.listLayoutKeys(target.slot),
  }
  return moveSlotEntry(deps.getLayout(), drawnKeys, {
    key: entry.key,
    from,
    to: target.slot,
    before: target.before,
  })
}

/** Writes the lists a drop changes, in one write; a drop that changes nothing writes nothing. */
const writeDrop = (entry: PressedEntry, target: SlotDropTarget, deps: LayoutDeps): void => {
  const change = getMoveChange(entry, entry.from, target, deps)
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

/**
 * The preview of a drag: the slots drawn with the entry where a drop would put it. It keeps the lists
 * the preview changed, and the slot the entry is drawn in now.
 */
function createPreview(deps: SlotDragDeps) {
  let pressed: PressedEntry | null = null
  let drawnIn: SlotName | null = null
  let lists: SlotLayoutChange = {}
  const clear = (): void => {
    lists = {}
    drawnIn = pressed?.from ?? null
    deps.showPreview?.(null)
  }
  return {
    clear,
    start: (entry: PressedEntry): void => {
      pressed = entry
      drawnIn = entry.from
      lists = {}
    },
    /** Draws the entry at the target; null puts it back where it was. */
    show: (target: SlotDropTarget | null): void => {
      if (!pressed || drawnIn === null) return
      if (!target) {
        if (Object.keys(lists).length > 0) clear()
        return
      }
      const change = getMoveChange(pressed, drawnIn, target, deps)
      if (Object.keys(change).length === 0) return
      lists = { ...lists, ...change }
      drawnIn = target.slot
      deps.showPreview?.(lists)
    },
  }
}

export function createSlotDrag(deps: SlotDragDeps): void {
  const root = document.documentElement
  const isCancelled = shallowRef(false)
  const clicks = createClickBlocker()
  const endDrag = (): void => root.removeAttribute(DRAGGING_ATTRIBUTE)
  const preview = createPreview(deps)
  const cancelDrag = (): void => {
    isCancelled.value = true
    endDrag()
    preview.clear()
  }

  const {
    dragging,
    at: dropTarget,
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
      // The drop is worked out from the layout as it was, not from the preview.
      preview.clear()
      if (target && !isCancelled.value) writeDrop(entry, target, deps)
    },
    cancel: () => {
      endDrag()
      preview.clear()
    },
  })

  watch(dropTarget, (target) => preview.show(isCancelled.value ? null : target), { flush: 'sync' })

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
      preview.start(entry)
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
}
