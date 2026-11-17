/** The names of the notes related to a note: which of them a side draws, in which order, and how many. */
import type { NoteGraph } from '@numen/plugin-sdk'
import type { NameRelation, NamesSide, RelatedName } from '../types'
import { WORDS } from '../words'

/** The relations a side names, in the order its names stand in. A sibling is never a name. */
const RELATIONS: Readonly<Record<NamesSide, readonly NameRelation[]>> = {
  above: ['parent'],
  below: ['child', 'jump', 'reference'],
}

/**
 * The names a side draws for the note of `noteGraph`, cut to `limit`. A note related in several relations is named once,
 * in the first relation of the order.
 */
export function listNames(noteGraph: NoteGraph, side: NamesSide, limit: number): RelatedName[] {
  const owner = noteGraph.focus.title || WORDS.untitled
  const named = new Set<string>()
  return RELATIONS[side]
    .flatMap((relation) =>
      noteGraph.related.filter((item) => item.relation === relation).map((item) => ({ item, relation })),
    )
    .filter(({ item }) => {
      if (named.has(item.path)) return false
      named.add(item.path)
      return true
    })
    .slice(0, limit)
    .map(({ item, relation }) => ({
      path: item.path,
      title: item.title,
      relation,
      ...(item.direction ? { direction: item.direction } : {}),
      label: WORDS.nameLabel(item.title || WORDS.untitled, owner, relation, item.direction),
    }))
}
