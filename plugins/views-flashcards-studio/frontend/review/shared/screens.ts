/**
 * Which screen the window is on, and what each screen holds.
 *
 * The screens form one ordered sequence and navigation goes back along it. What
 * a screen holds is declared once, with the screen; going back releases
 * everything held by the screens being left. What is released follows from the
 * navigation path, and no screen change carries a cleanup list of its own.
 */
import { ref } from 'vue'
import type { Ref } from 'vue'

/**
 * What each screen holds, by screen. A screen holding nothing has no entry here.
 */
export type ScreenCleanups<Name extends string> = Readonly<
  Partial<Record<Name, readonly (() => void)[]>>
>

/** Order is the screens from the first one in to the last, and never empty. */
export function useScreens<Name extends string>(
  order: readonly [Name, ...Name[]],
  cleanups: ScreenCleanups<Name>,
) {
  const on = ref<Name>(order[0]) as Ref<Name>

  /**
   * To a screen. Every screen standing after it lets go of what it holds, so a
   * person going back never meets what the screen they left was showing.
   */
  const goTo = (to: Name) => {
    for (const name of order.slice(order.indexOf(to) + 1)) {
      for (const cleanup of cleanups[name] ?? []) cleanup()
    }
    on.value = to
  }

  return { on, goTo }
}
