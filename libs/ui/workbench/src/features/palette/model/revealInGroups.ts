import { nextTick } from 'vue'

/** The handle of a rendered group, which can reveal one of its items. */
export interface PaletteGroupHandle {
  readonly reveal: (item: string) => void
}

/** Brings an item into view in whichever rendered group holds it, once the groups are drawn. */
export async function revealInGroups(
  groups: readonly PaletteGroupHandle[] | null,
  item: string,
): Promise<void> {
  await nextTick()
  for (const group of groups ?? []) group.reveal(item)
}
