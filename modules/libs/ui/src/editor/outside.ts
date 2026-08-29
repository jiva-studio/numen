/**
 * The three things the editor cannot know on its own.
 *
 * An address in the text is written for wherever that text lives, and
 * following one is the application's business. So is keeping the text: the
 * editor holds a document and has nowhere to put it.
 */
import { Facet } from '@codemirror/state'

type Resolve = (address: string) => string
type Open = (address: string) => void
type Save = () => void

/** What an address in the text becomes before the window loads it. */
export const resolving = Facet.define<Resolve, Resolve>({
  combine: (all) => all[0] ?? ((address) => address),
})

/** Where a drawn link goes when it is clicked. */
export const opening = Facet.define<Open, Open>({
  combine: (all) => all[0] ?? (() => {}),
})

/** Who is told that the person asked for the text to be kept now. */
export const saving = Facet.define<Save, Save>({
  combine: (all) => all[0] ?? (() => {}),
})
