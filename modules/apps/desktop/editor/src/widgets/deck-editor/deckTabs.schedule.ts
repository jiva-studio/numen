/**
 * Stencil listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { Move } from '../../shared/core'
import type { Cards, StencilSummary } from '../../shared/flashcards/cards'
import type { Presets } from '../preset-editor/core'
import type { openNotes } from '../note-editor/notes'
import { useDeckSchedule } from './scheduler'
import { sameOffers, stencilsOf } from './deck'

export function useDeckScheduleWiring(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  /** The stencils of the vault, as they were last listed. */
  const offers = shallowRef<readonly StencilSummary[]>([])
  /** Whether the last listing of the stencils answered. */
  let listedOk = true

  /** The stencils a card may be cut by, made again where the list changed. */
  const stencils = computed(() => stencilsOf(offers.value))

  /**
   * The stencils of the vault, asked for again. A list naming the same
   * stencils leaves the one held standing, so what is drawn under it stands
   * with it.
   */
  const lists = async (): Promise<void> => {
    try {
      const listed = (await cards.stencils()).stencils
      listedOk = true
      if (!sameOffers(offers.value, listed)) offers.value = listed
    } catch {
      listedOk = false
    }
  }

  /** The stencils asked for again, where the last listing did not answer. */
  const listsAgain = (): void => {
    if (!listedOk) void lists()
  }

  const scheduled = useDeckSchedule(presets, store)

  const changed = (paths: readonly string[], renamed: readonly Move[] = []): void => {
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
