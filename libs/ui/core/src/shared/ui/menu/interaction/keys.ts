/**
 * The keyboard, while a menu is open: which item it is on, and what each key
 * does with it.
 *
 * Tab moves within the items and wraps, which is what keeps the keyboard
 * names.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { stepTo } from '@/shared/lib/dom/roving'
import { canLandOn, type MenuItem } from '../items/item'
import { isLetter, jumpTo, NOTHING_TYPED, type Typeahead } from '../items/typeahead'

/** A rendered row, which takes keyboard focus on request. */
interface RowHandle {
  readonly focus: () => void
}

export interface MenuKeysState {
  /** Which item the keyboard is on, or -1 when it is on none. */
  readonly here: Ref<number>
  /** A rendered row, stored under the item it belongs to. */
  readonly holdRow: (item: string, row: unknown) => void
  /** The keyboard onto an item, or onto the menu itself where there is none. */
  readonly goTo: (pos: number) => void
  /** A key pressed inside the menu. */
  readonly onKey: (event: KeyboardEvent) => void
}

export interface MenuSubmenuActions {
  readonly onOpenSubmenu?: ((item: MenuItem) => void) | undefined
  readonly onDismissSubmenu?: (() => void) | undefined
}

/** The arrow keys that open or dismiss a submenu; true where one was taken. */
function onSubmenuKey(
  event: KeyboardEvent,
  item: MenuItem | undefined,
  actions: MenuSubmenuActions | undefined,
): boolean {
  if (event.key === 'ArrowRight' && actions?.onOpenSubmenu) {
    if (item && item.children && item.children.length > 0) {
      event.preventDefault()
      actions.onOpenSubmenu(item)
      return true
    }
  }
  if (event.key === 'ArrowLeft' && actions?.onDismissSubmenu) {
    event.preventDefault()
    actions.onDismissSubmenu()
    return true
  }
  return false
}

/** The keys that move the keyboard along the items; true where one was taken. */
function onStepKey(
  event: KeyboardEvent,
  stepForward: () => void,
  stepBack: () => void,
  step: (delta: number, from: number) => void,
): boolean {
  switch (event.key) {
    case 'ArrowDown':
      stepForward()
      return true
    case 'ArrowUp':
      stepBack()
      return true
    case 'Home':
      step(1, -1)
      return true
    case 'End':
      step(-1, 0)
      return true
    case 'Tab':
      if (event.shiftKey) stepBack()
      else stepForward()
      return true
    default:
      return false
  }
}

export function useMenuKeys(
  items: () => readonly MenuItem[],
  menu: Readonly<ShallowRef<HTMLElement | null>>,
  /** What the moment is, which the word being typed is carried on by. */
  now: () => number,
  submenuActions?: MenuSubmenuActions,
): MenuKeysState {
  const here = ref(-1)

  /** Each rendered row, stored under its item. */
  const renderedRows = new Map<string, RowHandle | null>()

  const holdRow = (item: string, row: unknown): void => {
    renderedRows.set(item, row as RowHandle | null)
  }

  const goTo = (pos: number): void => {
    here.value = pos
    const item = items()[pos]
    const selected = item ? renderedRows.get(item.id) : undefined
    if (selected) selected.focus()
    else menu.value?.focus()
  }

  /** The word being typed to jump by, which the next letter carries on. */
  let activeTypeahead: Typeahead = NOTHING_TYPED

  /** The keyboard onto the item a letter names, and nowhere where it names none. */
  const jump = (letter: string): void => {
    const jumped = jumpTo(items(), activeTypeahead, { letter, from: here.value, now: now() })
    activeTypeahead = jumped.typed
    if (jumped.at !== null) goTo(jumped.at)
  }

  const onKey = (event: KeyboardEvent): void => {
    const step = (delta: number, from: number): void => {
      event.preventDefault()
      goTo(stepTo(items(), { from, by: delta }, canLandOn))
    }
    const stepForward = (): void => step(1, here.value)
    // Counting back from no item is counting back from the first.
    const stepBack = (): void => step(-1, Math.max(here.value, 0))

    if (onSubmenuKey(event, items()[here.value], submenuActions)) return
    if (onStepKey(event, stepForward, stepBack, step)) return
    if (isLetter(event)) {
      event.preventDefault()
      jump(event.key)
    }
  }

  return { here, holdRow, goTo, onKey }
}
