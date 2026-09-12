/**
 * Which screen the window is on, and what each screen holds.
 *
 * The screens stand in one order and a person goes back along it. What a screen
 * took up is declared once, beside the screen; going back lets go of everything
 * the screens being left hold. So what is forgotten follows from where a person
 * went, and no screen change carries a list of its own to keep up to date.
 */
import { ref } from 'vue'
import type { Ref } from 'vue'

/**
 * What each screen holds, by the screen it belongs to. A screen holding nothing
 * says nothing here.
 */
export type Holdings<Name extends string> = Readonly<
  Partial<Record<Name, readonly (() => void)[]>>
>

/** Order is the screens from the first one in to the last, and never empty. */
export function useScreens<Name extends string>(
  order: readonly [Name, ...Name[]],
  holds: Holdings<Name>,
) {
  const on = ref<Name>(order[0]) as Ref<Name>

  /**
   * To a screen. Every screen standing after it lets go of what it holds, so a
   * person going back never meets what the screen they left was showing.
   */
  const goTo = (to: Name) => {
    for (const name of order.slice(order.indexOf(to) + 1)) {
      for (const forget of holds[name] ?? []) forget()
    }
    on.value = to
  }

  return { on, goTo }
}
