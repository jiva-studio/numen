/**
 * The Cmd/Ctrl+drag of the entries of the slots. A press with the modifier on an entry the plugins
 * mark starts a press; past the threshold it is a drag, and a drop on a marked slot writes the lists
 * the move changes in one write. While the drag goes on, a copy of the entry follows the pointer, the
 * entry itself stands dimmed where a drop would put it, and the entries around it slide aside.
 * Every other press is left to the page.
 */
import {
  computed,
  nextTick,
  onScopeDispose,
  shallowRef,
  watch,
  type ComputedRef,
  type Ref,
} from 'vue'
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
import { listEntryElements, measureEntries, playMoves } from '../lib/flip'

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

/** The attribute of the entry that stands where a drop would put the dragged one. */
export const PLACEHOLDER_ATTRIBUTE = 'data-numen-slot-placeholder'

/** What the layer of the drag draws: a copy of the dragged entry and where the pointer is. */
export interface SlotDragState {
  readonly copy: ComputedRef<SlotDragCopy | null>
  readonly position: Readonly<Ref<Position | null>>
}

/** A copy of the dragged entry, with the size the entry had. */
export interface SlotDragCopy {
  readonly node: Node
  readonly width: number
  readonly height: number
}

/** The entry a press holds: its layout key, its entry attribute, the slot it is drawn in and a copy. */
interface PressedEntry {
  readonly key: string
  readonly itemKey: string
  readonly from: SlotName
  readonly copy: SlotDragCopy
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
  const box = measureRect(entry)
  const copy = { node: entry.cloneNode(true), width: box.width, height: box.height }
  return { key, itemKey: entry.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '', from: found.slot, copy }
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

/** The drawn lists of every slot and the layout, as they were when the press began. */
interface PressSnapshot {
  readonly layout: SlotLayout
  readonly drawn: Partial<Record<SlotName, readonly string[]>>
}

/**
 * The preview of a drag: the slots drawn with the entry where a drop would put it, and the entries
 * around it sliding to their new places. Every change is worked out from the state at the press.
 */
function createPreview(deps: SlotDragDeps) {
  let pressed: PressedEntry | null = null
  let snapshot: PressSnapshot | null = null
  let shown = ''

  const markPlaceholder = (): void => {
    for (const element of listEntryElements()) element.removeAttribute(PLACEHOLDER_ATTRIBUTE)
    if (!pressed || shown === '') return
    const { itemKey } = pressed
    const placeholder = listEntryElements().find(
      (element) => element.getAttribute(SLOT_ENTRY_ATTRIBUTE) === itemKey,
    )
    placeholder?.setAttribute(PLACEHOLDER_ATTRIBUTE, '')
  }

  const draw = (change: SlotLayoutChange | null): void => {
    const key = change ? JSON.stringify(change) : ''
    if (key === shown) return
    shown = key
    const before = measureEntries()
    deps.showPreview?.(change)
    nextTick()
      .then(() => {
        playMoves(before, pressed?.itemKey ?? '')
        markPlaceholder()
      })
      .catch(() => undefined)
  }

  /** The lists the entry changes by going to the target; none when it stays where it is. */
  const getChange = (target: SlotDropTarget): SlotLayoutChange =>
    pressed && snapshot
      ? moveSlotEntry(snapshot.layout, snapshot.drawn, {
          key: pressed.key,
          from: pressed.from,
          to: target.slot,
          before: target.before,
        })
      : {}

  return {
    getChange,
    start: (entry: PressedEntry): void => {
      pressed = entry
      shown = ''
      snapshot = {
        layout: deps.getLayout(),
        drawn: Object.fromEntries(
          SLOT_NAMES.map((slot) => [slot, deps.listLayoutKeys(slot as SlotName)]),
        ),
      }
    },
    /** Draws the entry at the target; null puts it back where it was. */
    show: (target: SlotDropTarget | null): void => {
      const change = target ? getChange(target) : {}
      draw(Object.keys(change).length > 0 ? change : null)
    },
    clear: (): void => draw(null),
    /** Keeps the preview until the write ends, so the slots do not show the old layout in between. */
    hold: (write: Promise<void>): void => {
      write.catch(() => undefined).finally(() => draw(null))
    },
  }
}

export function createSlotDrag(deps: SlotDragDeps): SlotDragState {
  const root = document.documentElement
  const isCancelled = shallowRef(false)
  const clicks = createClickBlocker()
  const endDrag = (): void => root.removeAttribute(DRAGGING_ATTRIBUTE)
  const preview = createPreview(deps)
  const pressedCopy = shallowRef<SlotDragCopy | null>(null)
  const cancelDrag = (): void => {
    isCancelled.value = true
    endDrag()
    preview.clear()
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
      const change = target && !isCancelled.value ? preview.getChange(target) : {}
      // The settings store reports a failed write itself and reads the file again.
      if (Object.keys(change).length > 0) preview.hold(deps.writeLayout(change))
      else preview.clear()
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
      pressedCopy.value = entry.copy
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
  return { copy: computed(() => (isDrawn() ? pressedCopy.value : null)), position }
}
