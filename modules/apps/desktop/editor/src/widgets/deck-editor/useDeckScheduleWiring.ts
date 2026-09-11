/**
 * Stencil listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename } from '../../shared/core'
import type { Cards, StencilSummary } from '../../entities/deck/cards'
import type { Presets } from '../preset-editor/core'
import type { openNotes } from '../note-editor/notes'
import { useDeckSchedule } from './useDeckSchedule'
import { sameOffers, stencilsOf } from './deck'

export function useDeckScheduleWiring(
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
      listedOk = false
    }
  }

  const listsAgain = (): void => {
    if (!listedOk) void lists()
  }

  const scheduled = useDeckSchedule(presets, store)

  const changed = (paths: readonly string[], renamed: readonly PathRename[] = []): void => {
    for (const went of renamed) {
      scheduled.moved(went.from, went.to)
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
    changed,
  }
}
