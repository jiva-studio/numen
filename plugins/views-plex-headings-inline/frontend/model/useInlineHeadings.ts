/** The headings one node writes under itself: those of its note, read while the node is shown. */
import { computed, onScopeDispose, ref, shallowRef, watch, type ComputedRef } from 'vue'
import { createRequestGuard } from '@numen/ui'
import type { Caught, NoteHeading, VaultNotes } from '@numen/plugin-sdk'
import type { HeadingsSettings } from './settings'

/** What the headings are read with, what chooses one, and when the notes of the vault change. */
export interface InlineSource {
  readonly notes: Pick<VaultNotes, 'getHeadings'>
  readonly settings: HeadingsSettings
  readonly open: (path: string, line: number) => void
  readonly onVaultChange: (listener: () => void) => { dispose: () => void }
  readonly reportFailure: (error: Caught) => void
}

/** The first headings of the note at `path`, whether their first read has ended, and `choose`, which opens the note at a line. */
export function useInlineHeadings(
  source: InlineSource,
  path: () => string,
): {
  headings: ComputedRef<readonly NoteHeading[]>
  isLoaded: ComputedRef<boolean>
  choose: (line: number) => void
} {
  const found = shallowRef<readonly NoteHeading[]>([])
  const isRead = ref(false)
  const changes = ref(0)
  const guard = createRequestGuard()
  onScopeDispose(guard.close)

  const subscription = source.onVaultChange(() => changes.value++)
  onScopeDispose(subscription.dispose)

  watch(
    [path, changes],
    async () => {
      const mine = guard.acquireRequest()
      const current = path()
      try {
        const answer = await source.notes.getHeadings([current])
        if (mine.isCurrent) found.value = answer.get(current) ?? []
      } catch (error) {
        if (mine.isCurrent) source.reportFailure(error)
      } finally {
        if (mine.isCurrent) isRead.value = true
      }
    },
    { immediate: true },
  )

  const headings = computed(() => found.value.slice(0, source.settings.headingLimit.value))
  const choose = (line: number) => source.open(path(), line)

  return { headings, isLoaded: computed(() => isRead.value), choose }
}
