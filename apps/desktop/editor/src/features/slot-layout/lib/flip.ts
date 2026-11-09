/**
 * The motion of the entries of the slots when a drag moves one: each entry that changed place slides
 * from its old box to its new one. A reader who asks for less motion gets none, in the style of the
 * drag layer.
 */
import { measureRect } from '@numen/ui'
import { SLOT_ENTRY_ATTRIBUTE } from '@numen/plugin-host'

/** How long an entry takes to slide into its new place. */
const SLIDE_TRANSITION = 'transform 150ms ease-out'

/** The boxes of the entries drawn now, by the value of their entry attribute. */
export type EntryBoxes = ReadonlyMap<string, DOMRect>

export const listEntryElements = (): Element[] => [
  ...document.querySelectorAll(`[${SLOT_ENTRY_ATTRIBUTE}]`),
]

export const measureEntries = (): EntryBoxes =>
  new Map(
    listEntryElements().map((element) => [
      element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '',
      measureRect(element),
    ]),
  )

/** Moves the entry back to where it was and lets it slide to where it is drawn now. */
const slide = (element: HTMLElement, dx: number, dy: number): void => {
  element.style.transition = 'none'
  element.style.transform = `translate(${dx}px, ${dy}px)`
  // Reading the box makes the browser draw the start of the slide before the end is set.
  measureRect(element)
  element.style.transition = SLIDE_TRANSITION
  element.style.transform = ''
  element.addEventListener(
    'transitionend',
    () => {
      element.style.transition = ''
    },
    { once: true },
  )
}

/** Slides every entry, but the one named `except`, from its box in `before` to where it is drawn now. */
export const playMoves = (before: EntryBoxes, except: string): void => {
  for (const element of listEntryElements()) {
    const key = element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? ''
    const from = before.get(key)
    if (key === except || !from || !(element instanceof HTMLElement)) continue
    const to = measureRect(element)
    const dx = from.left - to.left
    const dy = from.top - to.top
    if (dx !== 0 || dy !== 0) slide(element, dx, dy)
  }
}
