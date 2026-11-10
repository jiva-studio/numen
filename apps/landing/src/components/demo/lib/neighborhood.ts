import { ENTROPY_PATH, type VaultNote } from './notes'

export type Relation = 'parent' | 'child' | 'jump' | 'sibling'

/**
 * A note drawn around the note in focus, and how it is joined to it. A sibling hangs from `through`, a
 * parent of the note in focus; `label` is the word written on the link.
 */
export interface Neighbor {
  readonly path: string
  readonly title: string
  readonly relation: Relation
  readonly through: string
  readonly label: string
}

export interface Neighborhood {
  readonly focus: { readonly path: string; readonly title: string }
  readonly related: readonly Neighbor[]
}

export type Notes = Readonly<Record<string, VaultNote>>
export type EdgeLabels = Readonly<Record<string, string>>

/** The most siblings that are drawn. */
const MOST_SIBLINGS = 2

export const getParentsOf = (notes: Notes, path: string): readonly string[] =>
  notes[path]?.parents ?? []

export const getChildrenOf = (notes: Notes, path: string): string[] =>
  Object.keys(notes).filter((other) => getParentsOf(notes, other).includes(path))

export function getJumpsOf(notes: Notes, path: string): string[] {
  const mine = notes[path]?.jumps ?? []
  const theirs = Object.keys(notes).filter((other) => (notes[other]?.jumps ?? []).includes(path))
  return [...new Set([...mine, ...theirs])]
}

function getSiblingsOf(notes: Notes, path: string): { path: string; through: string }[] {
  const result: { path: string; through: string }[] = []
  const seen = new Set<string>([
    path,
    ...getParentsOf(notes, path),
    ...getChildrenOf(notes, path),
    ...getJumpsOf(notes, path),
  ])
  for (const parent of getParentsOf(notes, path)) {
    for (const child of getChildrenOf(notes, parent)) {
      if (seen.has(child)) continue
      seen.add(child)
      result.push({ path: child, through: parent })
    }
  }
  return result
}

function getLabel(labels: EdgeLabels, relation: Relation, focus: string, path: string): string {
  if (relation === 'child') return labels[`${focus}->${path}`] ?? ''
  if (relation === 'sibling') return ''
  return labels[`${path}->${focus}`] ?? labels[`${focus}->${path}`] ?? ''
}

/** The note in focus with its parents, children, jumps and the first two siblings. */
export function getNeighborhood(notes: Notes, labels: EdgeLabels, focusPath: string): Neighborhood {
  const resolved = notes[focusPath] ? focusPath : ENTROPY_PATH
  const focus = notes[resolved]
  if (!focus) throw new Error(`the demo vault has no note at ${ENTROPY_PATH}`)
  const related: Neighbor[] = []
  const seen = new Set<string>([resolved])
  const add = (path: string, relation: Relation, through = '') => {
    const note = notes[path]
    if (seen.has(path) || !note) return
    seen.add(path)
    related.push({
      path,
      title: note.title,
      relation,
      through,
      label: getLabel(labels, relation, resolved, path),
    })
  }
  for (const path of getParentsOf(notes, resolved)) add(path, 'parent')
  for (const path of getChildrenOf(notes, resolved)) add(path, 'child')
  for (const path of getJumpsOf(notes, resolved)) add(path, 'jump')
  for (const one of getSiblingsOf(notes, resolved).slice(0, MOST_SIBLINGS)) {
    add(one.path, 'sibling', one.through)
  }
  return { focus: { path: resolved, title: focus.title }, related }
}
