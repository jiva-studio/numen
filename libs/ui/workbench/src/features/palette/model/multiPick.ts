/**
 * What the palette shows and emits for a step that asks for several items.
 *
 * Without `multi` the groups are the caller's and a choice is emitted as it was made.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { PaletteGroup } from '../lib/item'
import { CREATE_ID, getMultiGroup, type PaletteMultiProps } from '../lib/multi'

interface MultiPickProps {
  readonly groups: readonly PaletteGroup[]
  readonly multi?: PaletteMultiProps | undefined
}

interface MultiPickEmit {
  (event: 'choose', item: string, action: string): void
  (event: 'toggle', id: string): void
  (event: 'create', text: string): void
}

export interface MultiPick {
  readonly groups: ComputedRef<readonly PaletteGroup[]>
  readonly tellChoice: (item: string, action: string) => void
}

export function useMultiPick(
  props: MultiPickProps,
  typed: Ref<string>,
  emit: MultiPickEmit,
): MultiPick {
  const groups = computed(() =>
    props.multi ? [getMultiGroup(props.multi, typed.value)] : props.groups,
  )

  const tellChoice = (item: string, action: string): void => {
    if (!props.multi) emit('choose', item, action)
    else if (item === CREATE_ID) emit('create', typed.value.trim())
    else emit('toggle', item)
  }

  return { groups, tellChoice }
}
