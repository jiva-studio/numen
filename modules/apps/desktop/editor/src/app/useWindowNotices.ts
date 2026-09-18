/**
 * System and status notices displayed in the corner of the window.
 */
import { computed, type ComputedRef } from 'vue'
import type { Notice } from '@numen/ui'
import { cornerOf } from '@/shared/notices/corner'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowDisplay } from './useWindowDisplay'
import type { useSettings } from './useSettings'
import type { useVaults } from './useVaults'
import { WORDS as words } from '@/shared/words'

export interface WindowNoticesDeps {
  log: MessageLog
  window: ReturnType<typeof useWindowDisplay>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
}

/**
 * Computes window corner notices based on background tasks, logs, and vault state.
 */
export function useWindowNotices({
  log,
  window,
  settings,
  vaults,
}: WindowNoticesDeps): ComputedRef<readonly Notice[]> {
  return computed<readonly Notice[]>(() =>
    cornerOf(
      window.tasks.value,
      log.messages.value,
      {
        unwatched: window.unwatched.value,
        unread: window.error.value,
        lost: window.lost.value || settings.dressed.lost.value,
        isReading: window.isIndexing.value,
        hasNote: window.hasNote.value,
      },
      vaults.coverage(),
      words,
    ),
  )
}
