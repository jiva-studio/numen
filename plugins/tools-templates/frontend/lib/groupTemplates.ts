import type { CommandItem } from '@numen/plugin-sdk'

/** A note found under the templates folder. */
export interface FoundTemplate {
  readonly path: string
}

/** Turns the notes found under the templates folder into the items of the picker. */
export function createTemplateItems(
  _root: string,
  _found: readonly FoundTemplate[],
): readonly CommandItem[] {
  throw new Error('not implemented')
}
