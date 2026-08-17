/**
 * The two things the editor cannot know on its own.
 *
 * An address in the text is written for wherever that text lives, and
 * following one is the application's business.
 */
import { Facet } from '@codemirror/state'

type Resolve = (address: string) => string
type Open = (address: string) => void

/** What an address in the text becomes before the window loads it. */
export const resolving = Facet.define<Resolve, Resolve>({
  combine: (all) => all[0] ?? ((address) => address),
})

/** Where a drawn link goes when it is clicked. */
export const opening = Facet.define<Open, Open>({
  combine: (all) => all[0] ?? (() => {}),
})
