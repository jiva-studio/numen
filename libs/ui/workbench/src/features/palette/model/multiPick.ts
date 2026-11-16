/**
 * What the palette shows and emits for a step that asks for several items.
 *
 * Without `multi` the groups are the caller's and a choice is emitted as it was made.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import type { PaletteGroup } from '../lib/item'
import {
  CREATE_ID,
  getMultiGroup,
  orderMultiItems,
  type PaletteMultiItem,
  type PaletteMultiProps,
} from '../lib/multi'

interface MultiPickProps {
  readonly groups: readonly PaletteGroup[]
  readonly multi?: PaletteMultiProps | undefined
  readonly isOpen?: boolean
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
  /** The ids of the items in the order the palette showed them first; not reactive on purpose. */
  const ids: string[] = []
  let createdTitle: string | undefined
  let openedWith = props.isOpen

  const arrange = (multi: PaletteMultiProps): readonly PaletteMultiItem[] => {
    if (props.isOpen !== openedWith) {
      openedWith = props.isOpen
      ids.length = 0
      createdTitle = undefined
    }
    const isFirst = ids.length === 0
    if (isFirst) ids.push(...orderMultiItems(multi.items, multi.recent).map((item) => item.id))
    for (const item of multi.items) {
      const isKnown = ids.includes(item.id)
      if (isKnown && !isFirst) continue
      if (item.title === createdTitle) {
        createdTitle = undefined
        if (isKnown) ids.splice(ids.indexOf(item.id), 1)
        ids.unshift(item.id)
      } else if (!isKnown) ids.push(item.id)
    }
    return ids.flatMap((id) => multi.items.find((item) => item.id === id) ?? [])
  }

  const groups = computed(() =>
    props.multi ? [getMultiGroup(props.multi, arrange(props.multi), typed.value)] : props.groups,
  )

  const tellChoice = (item: string, action: string): void => {
    if (!props.multi) emit('choose', item, action)
    else if (item === CREATE_ID) {
      createdTitle = typed.value.trim()
      emit('create', createdTitle)
    } else emit('toggle', item)
  }

  return { groups, tellChoice }
}
