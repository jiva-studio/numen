/**
 * Card model listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename } from '@numen/plugin-sdk'
import type { openNotes } from '../../buffer'
import type { Cards, CardModelSummary, Presets } from '../../cards'
import { useDeckSchedule } from './useDeckSchedule'
import { areOffersEqual, createCardModelChoices } from '../lib/deck'

export function useDeckScheduleSync(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  const offers = shallowRef<readonly CardModelSummary[]>([])
  let listedOk = true

  const cardModels = computed(() => createCardModelChoices(offers.value))

  const listCardModels = async (): Promise<void> => {
    try {
      const items = (await cards.cardModels()).cardModels
      listedOk = true
      if (!areOffersEqual(offers.value, items)) offers.value = items
    } catch {
      // Listing card models failed.
      listedOk = false
    }
  }

  const listCardModelsAgain = (): void => {
    if (!listedOk) void listCardModels()
  }

  const scheduled = useDeckSchedule(presets, store)

  const applyPathChanges = (
    paths: readonly string[],
    // Stryker disable next-line ArrayDeclaration: an entry that is not a rename names no file, and every walk over renames skips it.
    renames: readonly PathRename[] = [],
  ): void => {
    for (const rename of renames) {
      scheduled.moveFile(rename.from, rename.to)
    }
    if (store.getOpenIds().length === 0) return
    void listCardModels()
    void scheduled.listing.listPresets()
    for (const one of store.getOpenIds()) void scheduled.refreshDeckPreset(store.getPath(one))
  }

  return {
    offers,
    cardModels,
    listCardModels,
    listCardModelsAgain,
    scheduled,
    applyPathChanges,
  }
}
