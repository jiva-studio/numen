/**
 * What the palette shows and emits for a step that asks for several items.
 *
 * Without `multi` the groups are the caller's and a choice is emitted as it was made.
 */
import { computed, type ComputedRef, type Ref, shallowRef, watch } from 'vue'
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

interface Opening {
  readonly ids: string[]
  createdTitle: string | undefined
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
  /**
   * The order of the items as the palette showed them first, and the title of the item the person asked
   * to create. Each opening of the palette starts a new one; what is inside is not reactive on purpose.
   */
  const opening = shallowRef<Opening>({ ids: [], createdTitle: undefined })

  watch(
    () => props.isOpen,
    () => {
      opening.value = { ids: [], createdTitle: undefined }
    },
    { flush: 'sync' },
  )

  const arrange = (multi: PaletteMultiProps): readonly PaletteMultiItem[] => {
    const current = opening.value
    const { ids } = current
    const isFirst = ids.length === 0
    if (isFirst) ids.push(...orderMultiItems(multi.items, multi.recent).map((item) => item.id))
    for (const item of multi.items) {
      const isKnown = ids.includes(item.id)
      if (isKnown && !isFirst) continue
      if (item.title === current.createdTitle) {
        current.createdTitle = undefined
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
      opening.value.createdTitle = typed.value.trim()
      emit('create', typed.value.trim())
    } else emit('toggle', item)
  }

  return { groups, tellChoice }
}
