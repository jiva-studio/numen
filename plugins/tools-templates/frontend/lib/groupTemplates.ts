import type { CommandItem } from '@numen/plugin-sdk'

/** A note a mask names, and the group the mask puts it in. */
export interface FoundTemplate {
  readonly path: string
  readonly group: string
}

const compare = (left: string, right: string): number => {
  if (left === right) return 0
  return left < right ? -1 : 1
}

const compareText = (left: string, right: string): number =>
  compare(left.toLowerCase(), right.toLowerCase())

const getTitle = (path: string): string => {
  const stem = (path.split('/').pop() ?? path).replace(/\.md$/i, '')
  const title = stem.replace(/\.template$/, '')
  return title === '' ? stem : title
}

/** Turns the notes the masks name into the items of the picker. */
export function createTemplateItems(found: readonly FoundTemplate[]): readonly CommandItem[] {
  return found
    .map(({ path, group }) => ({ path, title: getTitle(path), group }))
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
