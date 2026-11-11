import { nextTick, type Ref } from 'vue'

/** The handle of a rendered group, which can reveal one of its items. */
export interface PaletteGroupHandle {
  readonly reveal: (item: string) => void
}

/** Brings an item into view in whichever rendered group holds it, once the groups are drawn. */
export function useGroupReveal(groups: Readonly<Ref<PaletteGroupHandle[] | null>>): {
  reveal: (item: string) => Promise<void>
} {
  async function reveal(item: string): Promise<void> {
    await nextTick()
    for (const group of groups.value ?? []) group.reveal(item)
  }
  return { reveal }
}
