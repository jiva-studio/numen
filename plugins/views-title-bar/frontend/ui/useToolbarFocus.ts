/**
 * The keyboard in the buttons of the bar, as in a toolbar: one of them is in the tab order, the
 * arrow keys move between them and wrap, and Home and End reach the first and the last.
 */
import { onMounted, onUpdated, type Ref } from 'vue'
import { stepTo } from '@numen/ui'

const FOCUSABLE = '[data-title-bar-focusable]'

/** How far each key moves along the buttons; Home and End step from beyond the ends. */
const STEPS: Readonly<Record<string, (index: number) => { from: number; by: number }>> = {
  ArrowRight: (index) => ({ from: index, by: 1 }),
  ArrowLeft: (index) => ({ from: index, by: -1 }),
  Home: () => ({ from: -1, by: 1 }),
  End: () => ({ from: 0, by: -1 }),
}

export function useToolbarFocus(toolbar: Readonly<Ref<HTMLElement | null>>) {
  let current: HTMLElement | null = null

  const listButtons = (): HTMLElement[] => [
    ...(toolbar.value?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? []),
  ]

  /** Puts the button reached last, or else the first, in the tab order, and the others out of it. */
  function syncTabOrder(): void {
    const buttons = listButtons()
    if (!current || !buttons.includes(current)) current = buttons[0] ?? null
    for (const button of buttons) button.tabIndex = button === current ? 0 : -1
  }

  function onKeydown(event: KeyboardEvent): void {
    const step = STEPS[event.key]
    const buttons = listButtons()
    const index = buttons.indexOf(event.target as HTMLElement)
    if (!step || index < 0) return
    const next = buttons[stepTo(buttons, step(index), () => true)]
    if (!next) return
    event.preventDefault()
    current = next
    syncTabOrder()
    next.focus()
  }

  function onFocusin(event: FocusEvent): void {
    const target = event.target as HTMLElement
    if (!listButtons().includes(target)) return
    current = target
    syncTabOrder()
  }

  onMounted(syncTabOrder)
  onUpdated(syncTabOrder)

  return { onKeydown, onFocusin }
}
