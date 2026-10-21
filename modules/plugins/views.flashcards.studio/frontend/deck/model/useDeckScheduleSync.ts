/**
 * Stencil listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename, openNotes } from '@numen/plugin-sdk'
import type { Cards, StencilSummary, Presets } from '../../cards'
import { useDeckSchedule } from './useDeckSchedule'
import { areOffersEqual, createStencilChoices } from '../lib/deck'

export function useDeckScheduleSync(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  const offers = shallowRef<readonly StencilSummary[]>([])
  let listedOk = true

  const stencils = computed(() => createStencilChoices(offers.value))

  const listStencils = async (): Promise<void> => {
    try {
      const items = (await cards.stencils()).stencils
      listedOk = true
      if (!areOffersEqual(offers.value, items)) offers.value = items
    } catch {
      // Listing stencils failed.
      listedOk = false
    }
  }

  const listStencilsAgain = (): void => {
    if (!listedOk) void listStencils()
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
