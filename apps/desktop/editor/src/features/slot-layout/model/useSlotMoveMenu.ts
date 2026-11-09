/**
 * The right-click menu of the entries of the bars. A right click on an entry of a slot the user
 * arranges opens the menu at the pointer; a choice writes the lists the move changes in one write.
 * A click anywhere else is left to the page.
 */
import { computed, onScopeDispose, shallowRef, type ComputedRef } from 'vue'
import type { MenuItem, Position } from '@numen/ui'
import {
  SLOT_ATTRIBUTE,
  SLOT_ENTRY_ATTRIBUTE,
  isUserArrangedSlot,
  listUserArrangedSlots,
  type LayoutEntry,
  type SlotLayout,
  type SlotLayoutChange,
  type SlotName,
} from '@numen/plugin-host'
import { getMoveChange, getMoveMenu, type MoveMenu, type MoveMenuWords } from './moveMenu'

export interface SlotMoveMenuDeps {
  readonly findLayoutKey: (itemKey: string, slot: SlotName) => string | undefined
  readonly listEntries: (slot: SlotName) => readonly LayoutEntry[]
  readonly getSlotTitle: (slot: SlotName) => string
  readonly getLayout: () => SlotLayout
  readonly writeLayout: (change: SlotLayoutChange) => Promise<void>
  readonly words: MoveMenuWords
}

/** What the view of the menu draws: the items and where the pointer was. */
export interface SlotMoveMenuState {
  readonly items: readonly MenuItem[]
  readonly at: Position
}

export interface SlotMoveMenuControl {
  readonly menu: ComputedRef<SlotMoveMenuState | null>
  /** Chooses an item: an item that moves the entry writes the move and closes the menu. */
  readonly choose: (id: string) => void
  readonly dismiss: () => void
}

interface OpenMenu {
  readonly key: string
  readonly from: SlotName
  readonly at: Position
  readonly menu: MoveMenu
}

/** The entry under a right click: its item key and the slot that draws it; null for any other element. */
const findEntry = (target: EventTarget | null): { itemKey: string; slot: SlotName } | null => {
  if (!(target instanceof Element)) return null
  const entry = target.closest(`[${SLOT_ENTRY_ATTRIBUTE}]`)
  const slot = entry?.closest(`[${SLOT_ATTRIBUTE}]`)?.getAttribute(SLOT_ATTRIBUTE)
  const itemKey = entry?.getAttribute(SLOT_ENTRY_ATTRIBUTE)
  if (!itemKey || !slot || !isUserArrangedSlot(slot)) return null
  return { itemKey, slot }
}

export function useSlotMoveMenu(deps: SlotMoveMenuDeps): SlotMoveMenuControl {
  const open = shallowRef<OpenMenu | null>(null)

  const onContextMenu = (event: MouseEvent): void => {
    const entry = findEntry(event.target)
    if (!entry) return
    const key = deps.findLayoutKey(entry.itemKey, entry.slot)
    if (key === undefined) return
    event.preventDefault()
    open.value = {
      key,
      from: entry.slot,
      at: { x: event.clientX, y: event.clientY },
      menu: getMoveMenu({ ...deps, listSlots: listUserArrangedSlots }, deps.words, key),
    }
  }
  document.addEventListener('contextmenu', onContextMenu, true)
  onScopeDispose(() => document.removeEventListener('contextmenu', onContextMenu, true))

  const dismiss = (): void => {
    open.value = null
  }

  const choose = (id: string): void => {
    const current = open.value
    const target = current?.menu.targets.get(id)
    if (!current || !target) return
    dismiss()
    const change = getMoveChange(
      deps.getLayout(),
      (slot) => deps.listEntries(slot).map((entry) => entry.key),
      { key: current.key, from: current.from, target },
    )
    if (Object.keys(change).length === 0) return
    // The settings store reports a failed write itself and reads the file again.
    deps.writeLayout(change).catch(() => undefined)
  }

  return {
    menu: computed(() => (open.value ? { items: open.value.menu.items, at: open.value.at } : null)),
    choose,
    dismiss,
  }
}
