/**
 * The Cmd/Ctrl+drag of the entries of the bars. A press with the modifier on an entry the bar plugins
 * mark starts a press; past the threshold it is a drag, and a drop on a bar slot writes the lists the
 * move changes in one write. Every other press is left to the page.
 */
import { computed, onScopeDispose, shallowRef, type ComputedRef, type Ref } from 'vue'
import { browserClock, usePressDrag, type Clock, type Position } from '@numen/ui'
import {
  BAR_SLOTS,
  SLOT_ATTRIBUTE,
  SLOT_ENTRY_ATTRIBUTE,
  isBarSlot,
  moveSlotEntry,
  type BarSlotName,
  type SlotLayout,
  type SlotLayoutChange,
} from '@numen/plugin-host'
import { findDropPlace, type EntryBox } from '../lib/dropPlace'

/** The attribute the root element of the page carries while a bar entry is dragged. */
export const DRAGGING_ATTRIBUTE = 'data-numen-slot-dragging'

/** How far, in pixels, the pointer moves before a press becomes a drag. */
const DRAG_THRESHOLD = 4

export interface SlotDragDeps {
  readonly isMac: () => boolean
  readonly findLayoutKey: (itemKey: string) => string | undefined
  readonly listLayoutKeys: (slot: BarSlotName) => readonly string[]
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

/** What the layer of the drag draws: the title of the dragged entry, where it is, and the marker. */
export interface SlotDragState {
  readonly label: ComputedRef<string | null>
  readonly position: Readonly<Ref<Position | null>>
  readonly marker: ComputedRef<SlotDropMarker | null>
}

/** The entry a press holds: its layout key, the slot it is drawn in and its title. */
interface PressedEntry {
  readonly key: string
  readonly from: BarSlotName
  readonly label: string
}

interface SlotDropTarget {
  readonly slot: BarSlotName
  readonly before: string | undefined
  readonly marker: SlotDropMarker
}

const findBarSlot = (element: Element | null): { element: Element; slot: BarSlotName } | null => {
  const slotElement = element?.closest(`[${SLOT_ATTRIBUTE}]`)
  const slot = slotElement?.getAttribute(SLOT_ATTRIBUTE)
  if (!slotElement || !slot || !isBarSlot(slot)) return null
  return { element: slotElement, slot }
}

const getEntryLabel = (element: Element, key: string): string => {
  const named = element.hasAttribute('aria-label') ? element : element.querySelector('[aria-label]')
  const label = named?.getAttribute('aria-label') ?? element.textContent ?? ''
  return label.trim() || key
}

const hasDragModifier = (event: PointerEvent, isMac: boolean): boolean =>
  isMac ? event.metaKey : event.ctrlKey

type LayoutDeps = Pick<SlotDragDeps, 'findLayoutKey' | 'listLayoutKeys' | 'getLayout' | 'writeLayout'>

const findPressedEntry = (target: EventTarget | null, deps: LayoutDeps): PressedEntry | null => {
  if (!(target instanceof Element)) return null
  const entry = target.closest(`[${SLOT_ENTRY_ATTRIBUTE}]`)
  const bar = findBarSlot(entry)
  const key = deps.findLayoutKey(entry?.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '')
  if (!entry || !bar || key === undefined) return null
  return { key, from: bar.slot, label: getEntryLabel(entry, key) }
}

const listEntryBoxes = (slotElement: Element, dragged: string, deps: LayoutDeps): EntryBox[] =>
  [...slotElement.querySelectorAll(`[${SLOT_ENTRY_ATTRIBUTE}]`)].flatMap((element) => {
    const key = deps.findLayoutKey(element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '')
    if (key === undefined || key === dragged) return []
    const box = element.getBoundingClientRect()
    return [{ key, left: box.left, right: box.right }]
  })

/** The bar slot under the point and the place in it the entry goes to; null outside the bar slots. */
const findDropTarget = (
  entry: PressedEntry,
  point: Position,
  deps: LayoutDeps,
): SlotDropTarget | null => {
  const bar = findBarSlot(document.elementFromPoint(point.x, point.y))
  if (!bar) return null
  const box = bar.element.getBoundingClientRect()
  const place = findDropPlace(listEntryBoxes(bar.element, entry.key, deps), point.x, box.left)
  return {
    slot: bar.slot,
    before: place.before,
    marker: { x: place.x, top: box.top, height: box.height },
  }
}

/** Writes the lists a drop changes, in one write; a drop that changes nothing writes nothing. */
const writeDrop = (entry: PressedEntry, target: SlotDropTarget, deps: LayoutDeps): void => {
  const drawnKeys = Object.fromEntries(
    BAR_SLOTS.map((slot) => [slot, deps.listLayoutKeys(slot)]),
  ) as Record<BarSlotName, readonly string[]>
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
  readonly pointerdown: (event: PointerEvent) => void
  readonly keydown: (event: KeyboardEvent) => void
  readonly click: (event: MouseEvent) => void
}

/** Adds the listeners to the document in the capture phase, before any element of the page, until disposal. */
const listenInCapture = (listeners: DocumentListeners): void => {
  document.addEventListener('pointerdown', listeners.pointerdown, true)
  document.addEventListener('keydown', listeners.keydown, true)
  document.addEventListener('click', listeners.click, true)
  onScopeDispose(() => {
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

  const { dragging, at: dropTarget, position, startPress } = usePressDrag<PressedEntry, SlotDropTarget>({
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

  listenInCapture({
    pointerdown: (event: PointerEvent): void => {
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
      isCancelled.value = true
      endDrag()
    },
    click: clicks.onClick,
  })
  onScopeDispose(endDrag)

  const isDrawn = (): boolean => dragging.value?.hasMoved === true && !isCancelled.value
  return {
    label: computed(() => (isDrawn() ? (dragging.value?.item.label ?? null) : null)),
    position,
    marker: computed(() => (isDrawn() ? (dropTarget.value?.marker ?? null) : null)),
  }
}
