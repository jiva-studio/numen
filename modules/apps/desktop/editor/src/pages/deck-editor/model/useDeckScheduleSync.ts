/**
 * Stencil listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename } from '@/shared/paths'
import type { Cards, StencilSummary } from '@/entities/deck'
import type { openNotes } from '@/entities/note'
import type { Presets } from '@/entities/deck'
import { useDeckSchedule } from './useDeckSchedule'
import { sameOffers, stencilsOf } from '../lib/deck'

export function useDeckScheduleSync(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  const offers = shallowRef<readonly StencilSummary[]>([])
  let listedOk = true

  const stencils = computed(() => stencilsOf(offers.value))

  const listStencils = async (): Promise<void> => {
    try {
      const listed = (await cards.stencils()).stencils
      listedOk = true
      if (!sameOffers(offers.value, listed)) offers.value = listed
    } catch {
      // Listing stencils failed.
      listedOk = false
    }
  }

  const listStencilsAgain = (): void => {
    if (!listedOk) void listStencils()
  }

  const scheduled = useDeckSchedule(presets, store)

  const applyPathChanges = (paths: readonly string[], renames: readonly PathRename[] = []): void => {
    for (const went of renames) {
      scheduled.moveFile(went.from, went.to)
    }
    if (store.getOpenIds().length === 0) return
    void listStencils()
    void scheduled.listPresets()
    for (const one of store.getOpenIds()) void scheduled.refreshDeckPreset(store.getPath(one))
  }

  return {
    offers,
    stencils,
    listStencils,
    listStencilsAgain,
    scheduled,
    applyPathChanges,
  }
}
