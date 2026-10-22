/**
 * The keyboard, while a menu is open: which item it is on, and what each key
 * does with it.
 *
 * Tab moves within the items and wraps, which is what keeps the keyboard
 * names.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { stepTo } from '@/shared/lib/roving'
import { canLandOn, type MenuItem } from './item'
import { isLetter, jumpTo, NOTHING_TYPED, type Typeahead } from './typeahead'

/** A row standing now, which takes the keyboard when it is asked to. */
interface RowHandle {
  readonly focus: () => void
}

export interface MenuKeysState {
  /** Which item the keyboard is on, or -1 when it is on none. */
  readonly here: Ref<number>
  /** A row as it is drawn, held under the item it stands for. */
  readonly holdRow: (item: string, row: unknown) => void
  /** The keyboard onto an item, or onto the menu itself where there is none. */
  readonly goTo: (at: number) => void
  /** A key pressed inside the menu. */
  readonly onKey: (event: KeyboardEvent) => void
}

export function useMenuKeys(
  items: () => readonly MenuItem[],
  menu: Readonly<ShallowRef<HTMLElement | null>>,
  /** What the moment is, which the word being typed is carried on by. */
  now: () => number,
): MenuKeysState {
  const here = ref(-1)

  /** Each item as it is drawn, each under the item it stands for. */
  const renderedRows = new Map<string, RowHandle | null>()

  const holdRow = (item: string, row: unknown): void => {
    renderedRows.set(item, row as RowHandle | null)
  }

  const goTo = (at: number): void => {
    here.value = at
    const item = items()[at]
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
    const step = (by: number, from: number): void => {
      event.preventDefault()
      goTo(stepTo(items(), { from, by }, canLandOn))
    }
    const stepForward = (): void => step(1, here.value)
    // Counting back from no item is counting back from the first.
    const stepBack = (): void => step(-1, Math.max(here.value, 0))
    if (event.key === 'ArrowDown') stepForward()
    else if (event.key === 'ArrowUp') stepBack()
    else if (event.key === 'Home') step(1, -1)
    else if (event.key === 'End') step(-1, 0)
    else if (event.key === 'Tab') {
      if (event.shiftKey) stepBack()
      else stepForward()
    } else if (isLetter(event)) {
      event.preventDefault()
      jump(event.key)
    }
  }

  return { here, holdRow, goTo, onKey }
}
