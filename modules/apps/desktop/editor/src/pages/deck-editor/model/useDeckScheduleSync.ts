/**
 * Stencil listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename } from '@/shared/paths'
import type { Cards, StencilSummary } from '@/entities/deck'
import type { openNotes } from '@/entities/note'
import type { Presets } from '@/entities/deck'
import { useDeckSchedule } from './useDeckSchedule'
import { sameOffers, stencilsOf } from '../deck'

export function useDeckScheduleSync(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  const offers = shallowRef<readonly StencilSummary[]>([])
  let listedOk = true

  const stencils = computed(() => stencilsOf(offers.value))

  const lists = async (): Promise<void> => {
    try {
      const listed = (await cards.stencils()).stencils
      listedOk = true
      if (!sameOffers(offers.value, listed)) offers.value = listed
    } catch {
      // Listing stencils failed.
      listedOk = false
    }
  }

  const listsAgain = (): void => {
    if (!listedOk) void lists()
  }

  const scheduled = useDeckSchedule(presets, store)

  const applyPathChanges = (paths: readonly string[], renamed: readonly PathRename[] = []): void => {
    for (const went of renamed) {
      scheduled.moveFile(went.from, went.to)
    }
    if (store.all().length === 0) return
    void lists()
    void scheduled.listsPresets()
    for (const one of store.all()) void scheduled.asks(store.where(one))
  }

  return {
    offers,
    stencils,
    lists,
    listsAgain,
    scheduled,
    applyPathChanges,
  }
}
