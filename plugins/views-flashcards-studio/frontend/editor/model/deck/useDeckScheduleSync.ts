/**
 * Card model listing and schedule coordination for deck tabs.
 */
import { computed, shallowRef } from 'vue'
import type { PathRename } from '@numen/plugin-sdk'
import type { openNotes } from '../buffer/notes'
import type { Presets } from '../../../lib/presets'
import { useDeckSchedule } from './useDeckSchedule'
import { areCardModelChoicesEqual, createCardModelChoices } from '../../lib/deck/deck'
import type { Cards, CardModelSummary } from '../../../types'

export function useDeckScheduleSync(
  cards: Cards,
  presets: Presets,
  store: ReturnType<typeof openNotes>,
) {
  const summaries = shallowRef<readonly CardModelSummary[]>([])
  let listedOk = true

  const cardModels = computed(() => createCardModelChoices(summaries.value))

  const listCardModels = async (): Promise<void> => {
    try {
      const items = (await cards.cardModels()).cardModels
      listedOk = true
      if (!areCardModelChoicesEqual(summaries.value, items)) summaries.value = items
    } catch {
      // Listing card models failed.
      listedOk = false
    }
  }

  const listCardModelsAgain = (): void => {
    if (!listedOk) {
      listCardModels().catch(() => {
        listedOk = false
      })
    }
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
    listCardModels().catch(() => {
      listedOk = false
    })
    scheduled.listing.listPresets().catch(() => {
      listedOk = false
    })
    for (const one of store.getOpenIds()) {
      const path = store.getPath(one)
      scheduled.refreshDeckPreset(path).catch(() => scheduled.forgetFile(path))
    }
  }

  return {
    summaries,
    cardModels,
    listCardModels,
    listCardModelsAgain,
    scheduled,
    applyPathChanges,
  }
}
