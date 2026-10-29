/**
 * System and status notices displayed in the corner of the window.
 */
import { computed, type ComputedRef } from 'vue'
import type { Notice, NoticeQueue } from '@numen/shell'
import { getCornerNotices } from '@/shared/notices/corner'
import type { MessageLog } from '@/shared/notices/messages'
import type { useWindowDisplay } from '../window/useWindowDisplay'
import type { useSettings } from '../window/useSettings'
import type { useVaults } from '../window/useVaults'
import { WORDS as words } from '@/shared/words'

export interface WindowNoticesDeps {
  log: MessageLog
  /** Plugin notices, shown after the window's own. */
  plugins: NoticeQueue
  window: ReturnType<typeof useWindowDisplay>
  settings: ReturnType<typeof useSettings>
  vaults: ReturnType<typeof useVaults>
}

/**
 * Computes window corner notices based on background tasks, logs, and vault state, and
 * what the plugins show after them.
 */
export function useWindowNotices({
  log,
  plugins,
  window,
  settings,
  vaults,
}: WindowNoticesDeps): ComputedRef<readonly Notice[]> {
  return computed<readonly Notice[]>(() => [
    ...getCornerNotices([], log.messages.value, {
      state: {
        unwatched: window.unwatched.value,
        unread: window.error.value,
        lost: window.lost.value || settings.appearance.lost.value,
        isIndexing: window.isIndexing.value,
        hasNote: window.hasNote.value,
      },
      vault: vaults.coverage(),
      words,
    }),
    ...plugins.notices.value,
  ])
}
