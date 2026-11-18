/**
 * Reactive state of the Logs viewer tab: watches `host.work.logs`, keeps a bounded list of recent
 * rows (newest first), filters by severity level and search query, and tracks expanded rows.
 */
import { computed, ref, shallowRef, type ComputedRef, type Ref } from 'vue'
import type { Disposable, LogEntry, LogHost } from '@numen/plugin-sdk'
import { createLogRow, filterRows } from '../lib/format'
import type { LevelFilter, LogRow } from '../types'

const MAX_ROWS = 1_000

type LogsPhase = 'loading' | 'ready' | 'unavailable'

/** Reactive state and actions of a Logs tab. */
export interface LogsView {
  readonly phase: Readonly<Ref<LogsPhase>>
  readonly levelFilter: Readonly<Ref<LevelFilter>>
  readonly searchQuery: Readonly<Ref<string>>
  readonly visibleRows: ComputedRef<readonly LogRow[]>
  readonly expandedIds: Readonly<Ref<ReadonlySet<number>>>
  setLevelFilter(level: LevelFilter): void
  setSearchQuery(query: string): void
  toggleExpanded(id: number): void
}

/** Controllable log stream lifecycle attached to the tab state. */
export interface LogsController {
  readonly view: LogsView
  start(): void
  stop(): void
}

export function createLogsView(host: LogHost | undefined): LogsController {
  const phase = ref<LogsPhase>('loading')
  const allRows = shallowRef<readonly LogRow[]>([])
  const levelFilter = ref<LevelFilter>('all')
  const searchQuery = ref('')
  const expandedIds = shallowRef<ReadonlySet<number>>(new Set())
  let nextId = 1
  let subscription: Disposable | undefined

  const visibleRows = computed(() =>
    filterRows(allRows.value, levelFilter.value, searchQuery.value),
  )

  const appendBatch = (entries: readonly LogEntry[]) => {
    if (entries.length > 0) {
      const added = entries.map((entry) => createLogRow(nextId++, entry)).reverse()
      const combined = [...added, ...allRows.value]
      allRows.value = combined.length > MAX_ROWS ? combined.slice(0, MAX_ROWS) : combined
    }
    phase.value = 'ready'
  }

  return {
    view: {
      phase,
      levelFilter,
      searchQuery,
      visibleRows,
      expandedIds,
      setLevelFilter: (level) => {
        levelFilter.value = level
      },
      setSearchQuery: (query) => {
        searchQuery.value = query
      },
      toggleExpanded: (id) => {
        const next = new Set(expandedIds.value)
        if (next.has(id)) {
          next.delete(id)
        } else {
          next.add(id)
        }
        expandedIds.value = next
      },
    },
    start: () => {
      if (!host) {
        phase.value = 'unavailable'
        return
      }
      try {
        subscription = host.watch(appendBatch)
      } catch {
        phase.value = 'unavailable'
      }
    },
    stop: () => {
      subscription?.dispose()
      subscription = undefined
    },
  }
}
