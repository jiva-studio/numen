/**
 * The keyboard, while a menu is open: which item it is on, and what each key
 * does with it.
 *
 * Tab moves within the items and wraps, which is what keeps the keyboard
 * names.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { stepTo } from '@/shared/lib/dom/roving'
import type { Unvalidated } from '@/shared/lib/async/types'
import { canLandOn, type MenuItem } from '../items/item'
import { isLetter, jumpTo, NOTHING_TYPED, type Typeahead } from '../items/typeahead'

/** A rendered row, which takes keyboard focus on request. */
interface RowHandle {
  readonly focus: (options?: FocusOptions) => void
}

/** A rendered menu row as the row component exposes it. */
export interface MenuRowHandle extends RowHandle {
  readonly getElement: () => HTMLElement | null
}

/** Whether the value is a menu row component, not an element or null. */
export const isMenuRowHandle = (value: Unvalidated): value is MenuRowHandle =>
  typeof value === 'object' &&
  value !== null &&
  'focus' in value &&
  typeof value.focus === 'function' &&
  'getElement' in value &&
  typeof value.getElement === 'function'

export interface MenuKeysState {
  readonly here: Ref<number>
  readonly holdRow: (item: string, row: RowHandle | null) => void
  readonly goTo: (pos: number) => void
  readonly pointTo: (pos: number) => void
  readonly onKey: (event: KeyboardEvent) => void
}

export interface MenuSubmenuActions {
  readonly onOpenSubmenu?: ((item: MenuItem) => void) | undefined
  readonly onDismissSubmenu?: (() => void) | undefined
}

function onSubmenuKey(
  event: KeyboardEvent,
  item: MenuItem | undefined,
  actions: MenuSubmenuActions | undefined,
): void {
  if (event.key === 'ArrowRight' && actions?.onOpenSubmenu) {
    if (item && item.children && item.children.length > 0) {
      event.preventDefault()
      actions.onOpenSubmenu(item)
    }
  } else if (event.key === 'ArrowLeft' && actions?.onDismissSubmenu) {
    event.preventDefault()
    actions.onDismissSubmenu()
  }
}

function onStepKey(
  event: KeyboardEvent,
  stepForward: () => void,
  stepBack: () => void,
  step: (delta: number, from: number) => void,
): void {
  switch (event.key) {
    case 'ArrowDown':
      stepForward()
      return
    case 'ArrowUp':
      stepBack()
      return
    case 'Home':
      step(1, -1)
      return
    case 'End':
      step(-1, 0)
      return
    case 'Tab':
      if (event.shiftKey) stepBack()
      else stepForward()
      return
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

  const holdRow = (item: string, row: RowHandle | null): void => {
    renderedRows.set(item, row)
  }

  const goTo = (pos: number): void => {
    here.value = pos
    const item = items()[pos]
    const selected = item ? renderedRows.get(item.id) : undefined
    if (selected) selected.focus()
    else menu.value?.focus()
  }

  /** The pointer onto an item: it becomes the active row, without scrolling the menu. */
  const pointTo = (pos: number): void => {
    here.value = pos
    const item = items()[pos]
    if (item) renderedRows.get(item.id)?.focus({ preventScroll: true })
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

    onSubmenuKey(event, items()[here.value], submenuActions)
    onStepKey(event, stepForward, stepBack, step)
    if (isLetter(event)) {
      event.preventDefault()
      jump(event.key)
    }
  }

  return { here, holdRow, goTo, pointTo, onKey }
}
