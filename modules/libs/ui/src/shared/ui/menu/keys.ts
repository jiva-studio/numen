/**
 * The keyboard, while a menu is open: which item it is on, and what each key
 * does with it.
 *
 * Tab moves within the items and wraps, which is what keeps the keyboard
 * inside a menu that stands over the page. A letter jumps to the item it
 * names.
 */
import { ref, type Ref, type ShallowRef } from 'vue'
import { stepTo, type MenuItem } from './item'
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
  const drawn = new Map<string, RowHandle>()

  const holdRow = (item: string, row: unknown): void => {
    if (row) drawn.set(item, row as RowHandle)
    else drawn.delete(item)
  }

  const goTo = (at: number): void => {
    here.value = at
    const item = items()[at]
    const chosen = item ? drawn.get(item.id) : undefined
    if (chosen) chosen.focus()
    else menu.value?.focus()
  }

  /** The word being typed to jump by, which the next letter carries on. */
  let typed: Typeahead = NOTHING_TYPED

  /** The keyboard onto the item a letter names, and nowhere where it names none. */
  const jump = (letter: string): void => {
    const jumped = jumpTo(items(), typed, letter, here.value, now())
    typed = jumped.typed
    if (jumped.at !== null) goTo(jumped.at)
  }

  const onKey = (event: KeyboardEvent): void => {
    // Counting back from no item is counting back from the first.
    const step = (by: number, from = by < 0 ? Math.max(here.value, 0) : here.value): void => {
      event.preventDefault()
      goTo(stepTo(items(), from, by))
    }
    if (event.key === 'ArrowDown') step(1)
    else if (event.key === 'ArrowUp') step(-1)
    else if (event.key === 'Home') step(1, -1)
    else if (event.key === 'End') step(-1, 0)
    else if (event.key === 'Tab') step(event.shiftKey ? -1 : 1)
    else if (isLetter(event)) {
      event.preventDefault()
      jump(event.key)
    }
  }

  return { here, holdRow, goTo, onKey }
}
