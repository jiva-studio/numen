/**
 * The time the notice corner has been held by the user, and the timestamp that
 * cards are read against.
 *
 * Time with the pointer over the corner, with keyboard focus in it, or with the
 * window hidden does not count as reading time. That duration is tracked here;
 * nothing here renders anything.
 */
import { computed, ref, type ComputedRef, type Ref, type ShallowRef } from 'vue'

/** The attention state of the notice corner. */
export interface NoticeStackState {
  /** The moment itself. */
  readonly now: Ref<number>
  /** The moment a card is read against. */
  readonly read: ComputedRef<number>
  /** Takes the clock forward, and the held time with it. */
  readonly beat: () => void
  /** A pointer went over a card, and left the stack. */
  readonly onPointerOver: (event: PointerEvent) => void
  readonly onPointerOut: (event: PointerEvent) => void
  /** The keyboard came into the stack, and left it. */
  readonly onFocusIn: () => void
  readonly onFocusOut: (event: FocusEvent) => void
}

export function useNoticeStack(
  stack: Readonly<ShallowRef<HTMLElement | null>>,
  clock: () => number,
  isHidden: () => boolean,
): NoticeStackState {
  const now = ref(clock())

  /** Whether a pointer is on the stack, and whether the keyboard is in it. */
  const pointed = ref(false)
  const focused = ref(false)
  /** Whether the corner is being held. */
  const isHolding = (): boolean => pointed.value || focused.value || isHidden()
  /** How long it has been held for. */
  const holdDuration = ref(0)

  const readItem = computed(() => now.value - holdDuration.value)

  /** The card a pointer is on, for as long as that card is still there. */
  let on: Element | null = null

  /**
   * A card can be removed while the pointer is over it, and a browser does not
   * guarantee a boundary event for a removed element, so the pointer and focus
   * state is re-checked against the page on each beat.
   */
  const beat = (): void => {
    if (pointed.value && on !== null && !on.isConnected) {
      pointed.value = false
      on = null
    }
    if (focused.value && !stack.value?.contains(document.activeElement)) focused.value = false
    const current = clock()
    if (isHolding()) holdDuration.value += current - now.value
    now.value = current
  }

  /** Answered by the card the pointer went over, whatever inside it was under it. */
  const onPointerOver = (event: PointerEvent): void => {
    on = event.currentTarget as Element | null
    pointed.value = true
  }

  const onPointerOut = (event: PointerEvent): void => {
    const to = event.relatedTarget
    if (to instanceof Node && stack.value?.contains(to)) return
    pointed.value = false
    on = null
  }

  const onFocusIn = (): void => {
    focused.value = true
  }

  const onFocusOut = (event: FocusEvent): void => {
    const to = event.relatedTarget
    if (to instanceof Node && stack.value?.contains(to)) return
    focused.value = false
  }

  return { now, read: readItem, beat, onPointerOver, onPointerOut, onFocusIn, onFocusOut }
}
