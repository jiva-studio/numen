/**
 * System and status notices displayed in the corner of the window.
 */
import { computed, type ComputedRef } from 'vue'
import type { Notice } from '@numen/ui'
import { cornerOf } from '../shared/notices/corner'
import type { MessageLog } from '../shared/notices/messages'
import type { useWindowShowing } from './showing'
import type { useSettings } from './settings'
import type { useVaults } from './vaults'
import { WORDS as words } from '../shared/words'

export interface WindowNoticesDeps {
  log: MessageLog
  window: ReturnType<typeof useWindowShowing>
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
        unread: window.trouble.value,
        lost: window.lost.value || settings.dressed.lost.value,
        reading: window.indexing.value,
        holds: window.holds.value,
      },
      vaults.coverage(),
      words,
    ),
  )
}
