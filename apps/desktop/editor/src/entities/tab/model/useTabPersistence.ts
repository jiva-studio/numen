import { ref, watch, type Ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { SavedTabsApi } from '../api/saved-tabs'
import type { OpenTabs } from '../lib/tab'
import type { Caught } from '@/shared/api/errors'

const SAVE_STATE_DELAY_MS = 500

function hasSameReport(before: OpenTabs, next: OpenTabs): boolean {
  return (
    before.activeId === next.activeId &&
    before.layout === next.layout &&
    before.tabs.length === next.tabs.length &&
    before.tabs.every((tab, index) => {
      const other = next.tabs[index]
      return (
        other !== undefined &&
        tab.id === other.id &&
        tab.kind === other.kind &&
        tab.title === other.title &&
        tab.path === other.path &&
        tab.place === other.place
      )
    })
  )
}

function createTabWriter(write: SavedTabsApi['writeOpenTabs']) {
  let lastWritten: string | undefined
  let pending: { value: OpenTabs; json: string } | null = null
  let sending: Promise<void> | null = null

  const writeCurrent = async (current: { value: OpenTabs; json: string }): Promise<void> => {
    try {
      await write(current.value)
      lastWritten = current.json
    } catch (error) {
      lastWritten = undefined
      if (!pending) throw error
    }
  }
  const sendPending = async (): Promise<void> => {
    try {
      while (pending) {
        const current = pending
        pending = null
        if (current.json === lastWritten) continue
        await writeCurrent(current)
      }
    } finally {
      sending = null
    }
  }
  return (next: OpenTabs): Promise<void> => {
    const json = JSON.stringify(next)
    if (!sending && json === lastWritten) return Promise.resolve()
    pending = { value: next, json }
    sending ??= sendPending()
    return sending
  }
}

/** Reports visible changes immediately and saves private tab state after input settles. */
export function useTabPersistence(
  open: Readonly<Ref<OpenTabs>>,
  isWriting: Readonly<Ref<boolean>>,
  write: SavedTabsApi['writeOpenTabs'],
  reportError: (error: Caught) => void,
) {
  const save = createTabWriter(write)
  const isPaused = ref(false)
  watch(
    [open, isWriting, isPaused],
    ([next, enabled, paused], [before, wasEnabled, wasPaused], onCleanup) => {
      if (!enabled || paused) return
      const writeSnapshot = () => runInBackground(save(next), reportError)
      if (before && wasEnabled && !wasPaused && hasSameReport(before, next)) {
        const timer = setTimeout(writeSnapshot, SAVE_STATE_DELAY_MS)
        onCleanup(() => clearTimeout(timer))
      } else {
        writeSnapshot()
      }
    },
    { immediate: true },
  )

  const flush = async (shouldPause = false): Promise<void> => {
    while (isWriting.value && !isPaused.value) {
      const current = open.value
      await save(current)
      if (current === open.value) break
    }
    if (shouldPause) isPaused.value = true
  }
  return {
    flush: () => flush(),
    pause: () => flush(true),
    resume: () => {
      isPaused.value = false
    },
  }
}
