/**
 * How long a person has had the corner in hand, and the moment cards are read
 * against.
 *
 * Time spent with the corner under a pointer, with the keyboard in it, or away
 * from the window entirely, is not time spent reading it. What that comes to is
 * kept here; nothing here draws anything.
 */
import { computed, ref, type ComputedRef, type Ref, type ShallowRef } from 'vue'

/** What the corner keeps of a person's attention. */
export interface NoticeStackState {
  /** The moment itself. */
  readonly now: Ref<number>
  /** The moment a card is read against. */
  readonly read: ComputedRef<number>
  /** Takes the clock forward, and the held time with it. */
  readonly beat: () => void
  /** A pointer went over a card, and left the stack. */
  readonly enters: (event: PointerEvent) => void
  readonly leaves: (event: PointerEvent) => void
  /** The keyboard came into the stack, and left it. */
  readonly holds: () => void
  readonly lets: (event: FocusEvent) => void
}

export function useNoticeStack(
  stack: Readonly<ShallowRef<HTMLElement | null>>,
  clock: () => number,
  hidden: () => boolean,
): NoticeStackState {
  const now = ref(clock())

  /** Whether a pointer is on the stack, and whether the keyboard is in it. */
  const pointed = ref(false)
  const focused = ref(false)
  /** Whether the corner is being held. */
  const holding = (): boolean => pointed.value || focused.value || hidden()
  /** How long it has been held for. */
  const heldFor = ref(0)

  const read = computed(() => now.value - heldFor.value)

  /** The card a pointer is on, for as long as that card is still there. */
  let on: Element | null = null

  /**
   * A card is taken out from under whatever was on it, and a browser owes
   * nothing about the boundary event for one that has gone, so what holds the
   * corner is asked of the page each time.
   */
  const beat = (): void => {
    if (pointed.value && on !== null && !on.isConnected) {
      pointed.value = false
      on = null
    }
    if (focused.value && !stack.value?.contains(document.activeElement)) focused.value = false
    const at = clock()
    if (holding()) heldFor.value += at - now.value
    now.value = at
  }

  /** Answered by the card the pointer went over, whatever inside it was under it. */
  const enters = (event: PointerEvent): void => {
    on = event.currentTarget as Element | null
    pointed.value = true
  }

  const leaves = (event: PointerEvent): void => {
    const to = event.relatedTarget
    if (to instanceof Node && stack.value?.contains(to)) return
    pointed.value = false
    on = null
  }

  const holds = (): void => {
    focused.value = true
  }

  const lets = (event: FocusEvent): void => {
    const to = event.relatedTarget
    if (to instanceof Node && stack.value?.contains(to)) return
    focused.value = false
  }

  return { now, read, beat, enters, leaves, holds, lets }
}
