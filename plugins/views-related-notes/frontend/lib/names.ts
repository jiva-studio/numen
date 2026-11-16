/** The names of the notes related to a note: which of them a side draws, in which order, and how many. */
import type { NoteGraph, NoteRelation, RelatedNote } from '@numen/plugin-sdk'
import type { NamesSide, RelatedName } from '../types'
import { WORDS } from '../words'

/** The relations a side names, in the order its names stand in. A sibling is never a name. */
const RELATIONS: Readonly<Record<NamesSide, readonly NoteRelation[]>> = {
  above: ['parent'],
  below: ['child', 'jump', 'reference'],
}

/**
 * The names a side draws for the note of `noteGraph`, cut to `limit`. A note related in several relations is named once,
 * in the first relation of the order. A note related to a related note of this note is not named.
 */
export function listNames(noteGraph: NoteGraph, side: NamesSide, limit: number): RelatedName[] {
  const owner = noteGraph.focus.title || WORDS.untitled
  const own = noteGraph.related.filter((item) => !item.relatedTo)
  const ordered: RelatedNote[] = RELATIONS[side].flatMap((relation) =>
    own.filter((item) => item.relation === relation),
  )
  const named = new Set<string>()
  return ordered
    .filter((item) => {
      if (named.has(item.path)) return false
      named.add(item.path)
      return true
    })
    .slice(0, limit)
    .map((item) => ({
      path: item.path,
      title: item.title,
      label: WORDS.nameLabel(item.title || WORDS.untitled, owner),
    }))
}
