import type { CommandItem } from '@numen/plugin-sdk'

/** A note found under the templates folder. */
export interface FoundTemplate {
  readonly path: string
}

const compare = (left: string, right: string): number => {
  if (left === right) return 0
  return left < right ? -1 : 1
}

const compareText = (left: string, right: string): number =>
  compare(left.toLowerCase(), right.toLowerCase())

const getTitle = (path: string): string =>
  (path.split('/').pop() ?? path).replace(/\.md$/i, '')

const getGroup = (root: string, path: string): string => {
  const inside = path.slice(root.length + 1)
  return inside.slice(0, Math.max(inside.lastIndexOf('/'), 0))
}

/** Turns the notes found under the templates folder into the items of the picker. */
export function createTemplateItems(
  root: string,
  found: readonly FoundTemplate[],
): readonly CommandItem[] {
  return found
    .map(({ path }) => ({ path, title: getTitle(path), group: getGroup(root, path) }))
    .sort(
      (left, right) =>
        compareText(left.group, right.group) ||
        compareText(left.title, right.title) ||
        compare(left.path, right.path),
    )
    .map(({ path, title, group }) =>
      group === '' ? { id: path, title } : { id: path, title, group },
    )
}
