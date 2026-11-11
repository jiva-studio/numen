import { nextTick, watch, type Ref } from 'vue'

/** Puts the keyboard in the rename field once it is drawn, when renaming starts. */
export function useRenameFocus(
  isRenaming: () => boolean,
  field: Readonly<Ref<{ focus(): void } | null>>,
): void {
  watch(isRenaming, async (on) => {
    if (!on) return
    await nextTick()
    field.value?.focus()
  })
}
