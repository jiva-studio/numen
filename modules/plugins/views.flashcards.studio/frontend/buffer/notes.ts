/**
 * Open deck and stencil buffer store for flashcards studio.
 */
import { shallowReactive } from 'vue'
import type { PathRename } from '@numen/plugin-sdk'
import type {
  ConflictWords,
  NoteBaseline,
  NoteErrorCode,
  Notes,
  OpenNote,
  OpenNotesOptions,
  State,
} from './types'
import { convertErrorCode, STALE_CONFLICT } from './words'

interface BufferTab {
  readonly id: string
  path: string
  filePath: string
  body: string
  state: State
  error: NoteErrorCode | null
  baseline: NoteBaseline | null
  keepOverride: boolean
}

export function openNotes(notes: Notes, options?: OpenNotesOptions) {
  const tabs = new Map<string, BufferTab>()
  const timers = new Map<string, ReturnType<typeof setTimeout>>()
  const firstChangeTime = new Map<string, number>()

  const debounceMs = options?.limits?.debounceMs ?? 1000
  const maxWaitMs = options?.limits?.maxWaitMs ?? 5000

  function clearTimer(id: string): void {
    const timer = timers.get(id)
    if (timer) {
      clearTimeout(timer)
      timers.delete(id)
    }
    firstChangeTime.delete(id)
  }

  function scheduleFlush(id: string): void {
    const existing = timers.get(id)
    if (existing) {
      clearTimeout(existing)
    }
    const now = Date.now()
    if (!firstChangeTime.has(id)) {
      firstChangeTime.set(id, now)
    }
    const first = firstChangeTime.get(id)!
    const remaining = Math.max(0, maxWaitMs - (now - first))
    const wait = Math.min(debounceMs, remaining)

    const timer = setTimeout(() => {
      clearTimer(id)
      void writeTab(id)
    }, wait)
    timers.set(id, timer)
  }

  async function writeTab(id: string): Promise<void> {
    const tab = tabs.get(id)
    if (!tab || tab.state !== 'unsaved') return
    clearTimer(id)
    tab.state = 'saving'
    const observed = tab.keepOverride ? null : tab.baseline
    try {
      const res = await notes.write(tab.path, tab.body, observed)
      if (!tabs.has(id)) return
      if (res.ok) {
        tab.state = 'clean'
        tab.error = null
        tab.keepOverride = false
        tab.filePath = res.value.at ?? tab.filePath
        tab.baseline = { prose: tab.body, fingerprint: res.value.at ?? '' }
      } else {
        if (res.error === 'changed') {
          tab.state = 'stale'
        } else {
          tab.state = 'error'
          tab.error = convertErrorCode(res.error)
        }
      }
    } catch {
      if (!tabs.has(id)) return
      tab.state = 'error'
      tab.error = 'unreachable'
    }
  }

  function open(id: string, path: string): void {
    const tab: BufferTab = shallowReactive({
      id,
      path,
      filePath: path,
      body: '',
      state: 'loading',
      error: null,
      baseline: null,
      keepOverride: false,
    })
    tabs.set(id, tab)
    void notes
      .read(path)
      .then((res) => {
        if (!tabs.has(id)) return
        if (res.ok) {
          tab.body = res.value.body
          tab.filePath = res.value.at ?? path
          tab.baseline = { prose: res.value.body, fingerprint: res.value.at ?? '' }
          tab.state = 'clean'
          tab.error = null
        } else {
          tab.state = 'error'
          tab.error = convertErrorCode(res.error)
        }
      })
      .catch(() => {
        if (!tabs.has(id)) return
        tab.state = 'error'
        tab.error = 'unreachable'
      })
  }

  function getOpenIds(): string[] {
    return Array.from(tabs.keys())
  }

  function getPath(id: string): string {
    return tabs.get(id)?.path ?? ''
  }

  function getFilePath(id: string): string {
    return tabs.get(id)?.filePath ?? tabs.get(id)?.path ?? ''
  }

  function getOpenNote(id: string): OpenNote {
    const tab = tabs.get(id)
    if (!tab) {
      return { path: '', body: '', state: 'loading', error: null }
    }
    return {
      path: tab.path,
      body: tab.body,
      state: tab.state,
      error: tab.error,
    }
  }

  function setBody(id: string, body: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    tab.body = body
    if (tab.state === 'clean' || tab.state === 'saving') {
      tab.state = 'unsaved'
    }
  }

  function saveBody(id: string, body: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    tab.body = body
    if (tab.state === 'clean' || tab.state === 'saving') {
      tab.state = 'unsaved'
    }
    scheduleFlush(id)
  }

  async function flushPending(id: string): Promise<void> {
    await writeTab(id)
  }

  async function flush(): Promise<void> {
    const pending = Array.from(tabs.keys()).map((id) => writeTab(id))
    await Promise.all(pending)
  }

  function keep(id: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    tab.keepOverride = true
    tab.state = 'unsaved'
    void writeTab(id)
  }

  function take(id: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    void notes
      .read(tab.path)
      .then((res) => {
        if (!tabs.has(id)) return
        if (res.ok) {
          tab.body = res.value.body
          tab.filePath = res.value.at ?? tab.path
          tab.baseline = { prose: res.value.body, fingerprint: res.value.at ?? '' }
          tab.state = 'clean'
          tab.error = null
          tab.keepOverride = false
          options?.onReplaced?.(tab.path)
        } else {
          tab.state = 'error'
          tab.error = convertErrorCode(res.error)
        }
      })
      .catch(() => {
        if (!tabs.has(id)) return
        tab.state = 'error'
        tab.error = 'unreachable'
      })
  }

  function stale(id: string): ConflictWords | null {
    const tab = tabs.get(id)
    return tab && tab.state === 'stale' ? STALE_CONFLICT : null
  }

  async function close(id: string): Promise<boolean> {
    const tab = tabs.get(id)
    if (!tab) return true
    if (tab.state === 'stale') return false
    clearTimer(id)
    tabs.delete(id)
    return true
  }

  function discard(id: string): void {
    clearTimer(id)
    tabs.delete(id)
  }

  function applyPathChanges(paths: readonly string[], renames: readonly PathRename[] = []): void {
    for (const rename of renames) {
      for (const tab of tabs.values()) {
        if (tab.path === rename.from) {
          tab.path = rename.to
          tab.filePath = rename.to
        }
      }
    }
    const notifyPathChange = (tab: BufferTab): void => {
      if (tab.state === 'unsaved') {
        tab.state = 'stale'
      } else if (tab.state === 'clean') {
        void notes.read(tab.path).then((res) => {
          if (res.ok) {
            tab.body = res.value.body
            tab.baseline = { prose: res.value.body, fingerprint: res.value.at ?? '' }
            tab.filePath = res.value.at ?? tab.filePath
            options?.onReplaced?.(tab.path)
          }
        })
      }
    }

    for (const path of paths) {
      for (const tab of tabs.values()) {
        if (tab.path === path) {
          notifyPathChange(tab)
        }
      }
    }
  }

  return {
    open,
    close,
    discard,
    getOpenIds,
    getPath,
    getFilePath,
    getOpenNote,
    setBody,
    saveBody,
    flushPending,
    flush,
    keep,
    take,
    stale,
    applyPathChanges,
  }
}
