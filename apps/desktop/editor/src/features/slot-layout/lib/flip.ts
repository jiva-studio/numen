/**
 * The motion of the entries of the slots when a drag moves one: each entry that changed place slides
 * from its old box to its new one.
 */
import { SLOT_ENTRY_ATTRIBUTE } from '@numen/plugin-host'

/** How long an entry takes to slide into its new place, in milliseconds. */
const SLIDE_DURATION = 150

/** The boxes of the entries drawn now, by the value of their entry attribute. */
export type EntryBoxes = ReadonlyMap<string, DOMRect>

export const listEntryElements = (): Element[] => [
  ...document.querySelectorAll(`[${SLOT_ENTRY_ATTRIBUTE}]`),
]

export const measureEntries = (): EntryBoxes =>
  new Map(
    listEntryElements().map((element) => [
      element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? '',
      element.getBoundingClientRect(),
    ]),
  )

/** Slides every entry, but the one named `except`, from its box in `before` to where it is drawn now. */
export const playMoves = (before: EntryBoxes, except: string): void => {
  if (globalThis.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
  for (const element of listEntryElements()) {
    const key = element.getAttribute(SLOT_ENTRY_ATTRIBUTE) ?? ''
    const from = before.get(key)
    if (key === except || !from || typeof element.animate !== 'function') continue
    const to = element.getBoundingClientRect()
    const dx = from.left - to.left
    const dy = from.top - to.top
    if (dx === 0 && dy === 0) continue
    element.animate(
      [{ transform: `translate(${dx}px, ${dy}px)` }, { transform: 'none' }],
      { duration: SLIDE_DURATION, easing: 'ease-out' },
    )
  }
}
