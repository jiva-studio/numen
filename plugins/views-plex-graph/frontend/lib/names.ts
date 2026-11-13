/** The names of the notes related to each related note, as the plex draws them. */
import type {
  NoteGraph,
  NoteRelation,
  PlexName,
  PlexNames,
  ReferenceMode,
  RelatedNote,
} from '../types'
import { isReferenceAdmitted } from './picture'

/** The order the names of a note stand in, parents first. A sibling is never a name. */
const NAME_ORDER: readonly NoteRelation[] = ['parent', 'child', 'jump', 'reference']

const NAME_PREFIX = 'name:'

/** The items that can be names, by the related note they belong to, in arrival order. */
function groupByOwner(
  noteGraph: NoteGraph,
  references: ReferenceMode,
): ReadonlyMap<string, readonly RelatedNote[]> {
  const owners = new Map<string, RelatedNote[]>()
  for (const item of noteGraph.related) {
    if (!item.relatedTo || !NAME_ORDER.includes(item.relation)) continue
    if (item.path === item.relatedTo) continue
    if (item.relation === 'reference' && !isReferenceAdmitted(item, references)) continue
    owners.set(item.relatedTo, [...(owners.get(item.relatedTo) ?? []), item])
  }
  return owners
}

/** The items in the order of the names, each note once: in the first relation it stands in. */
function listDistinct(items: readonly RelatedNote[]): RelatedNote[] {
  const kept = NAME_ORDER.flatMap((relation) => items.filter((item) => item.relation === relation))
  const named = new Set<string>()
  return kept.filter((item) => {
    if (named.has(item.path)) return false
    named.add(item.path)
    return true
  })
}

/**
 * The names of the notes related to each related note: its parents over it, and its children, jumps and references under it,
 * cut to `limit` in that order. A reference is a name when the references setting admits it, as for a node. A name has the id `token` gives its path.
 * A note related to the same owner in several relations is named once, in the first relation of the order. A note with no names has no entry.
 */
export function getNames(
  noteGraph: NoteGraph,
  limit: number,
  token: (path: string) => string,
  references: ReferenceMode = 'both',
): ReadonlyMap<string, PlexNames> {
  const toName = (item: RelatedNote): PlexName => ({ id: token(item.path), title: item.title })
  const found = new Map<string, PlexNames>()
  for (const [owner, items] of groupByOwner(noteGraph, references)) {
    const kept = listDistinct(items).slice(0, limit)
    found.set(token(owner), {
      above: kept.filter((item) => item.relation === 'parent').map(toName),
      below: kept.filter((item) => item.relation !== 'parent').map(toName),
    })
  }
  return found
}

/**
 * The id a name is handed to the plex under: the path of its note, in a word of its own, so that
 * choosing it reaches the note whether or not that note is drawn as a node.
 */
export const getNameToken = (path: string): string => `${NAME_PREFIX}${path}`

/** The path of the note a name stands for, and empty for an id that is not a name. */
export const getNamePath = (id: string): string =>
  id.startsWith(NAME_PREFIX) ? id.slice(NAME_PREFIX.length) : ''
