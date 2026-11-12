/**
 * Open deck and card model buffer store for flashcards studio.
 */
import { getNow } from '@numen/ui'
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
} from '../../lib/buffer/types'
import { convertErrorCode, STALE_CONFLICT } from '../../lib/buffer/words'

/** The write errors that the next edit retries, as in the notes editor. */
const RETRYABLE_ERRORS: readonly NoteErrorCode[] = ['tooLarge', 'bodyUnwritable', 'unreachable']

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

  function markUnreachable(id: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    tab.state = 'error'
    tab.error = 'unreachable'
  }

  function scheduleFlush(id: string): void {
    const existing = timers.get(id)
    if (existing) {
      clearTimeout(existing)
    }
    const now = getNow()
    if (!firstChangeTime.has(id)) {
      firstChangeTime.set(id, now)
    }
    const first = firstChangeTime.get(id)!
    const remaining = Math.max(0, maxWaitMs - (now - first))
    const wait = Math.min(debounceMs, remaining)

    const timer = setTimeout(() => {
      clearTimer(id)
      writeTab(id).catch(() => markUnreachable(id))
    }, wait)
    timers.set(id, timer)
  }

  function finishWrite(tab: BufferTab, body: string, path: string | undefined): void {
    // An edit made while the write was in flight stays unsaved, with its own flush scheduled.
    if (tab.state === 'saving') tab.state = 'clean'
    tab.error = null
    tab.keepOverride = false
    tab.filePath = path ?? tab.filePath
    tab.baseline = { prose: body, stat: path ?? '' }
  }

  function failWrite(tab: BufferTab, code: string): void {
    if (code === 'changed') {
      tab.state = 'stale'
      return
    }
    tab.state = 'error'
    tab.error = convertErrorCode(code)
  }

  async function writeTab(id: string): Promise<void> {
    const tab = tabs.get(id)
    if (!tab || tab.state !== 'unsaved') return
    clearTimer(id)
    tab.state = 'saving'
    const observed = tab.keepOverride ? null : tab.baseline
    const body = tab.body
    try {
      const res = await notes.write(tab.path, body, observed)
      if (!tabs.has(id)) return
      if (res.ok) finishWrite(tab, body, res.value.at)
      else failWrite(tab, res.error)
    } catch {
      if (tabs.has(id)) markUnreachable(id)
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
    notes
      .read(path)
      .then((res) => {
        if (!tabs.has(id)) return
        if (res.ok) {
          tab.body = res.value.body
          tab.filePath = res.value.at ?? path
          tab.baseline = { prose: res.value.body, stat: res.value.at ?? '' }
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
      return { path: '', body: '', state: 'loading', error: null, isRead: false }
    }
    return {
      path: tab.path,
      body: tab.body,
      state: tab.state,
      error: tab.error,
      isRead: tab.baseline !== null,
    }
  }

  /** Replaces the text of a tab, and writes it once the edits pause. */
  function setBody(id: string, body: string): void {
    const tab = tabs.get(id)
    // An edit is refused until a read returned the body: a pending read would replace it, and after a failed read there is no file version to write it against.
    if (!tab || tab.baseline === null) return
    tab.body = body
    if (tab.state === 'error' && tab.error !== null && RETRYABLE_ERRORS.includes(tab.error)) {
      tab.state = 'unsaved'
      tab.error = null
    }
    if (tab.state === 'clean' || tab.state === 'saving') {
      tab.state = 'unsaved'
    }
    scheduleFlush(id)
  }

  async function flushPending(id: string): Promise<void> {
    await writeTab(id)
  }

  /** A tab whose text an error stopped from being written. */
  const isUnwritten = (tab: BufferTab): boolean =>
    tab.state === 'error' && tab.baseline !== null && tab.body !== tab.baseline.prose

  /** Writes every tab, trying again those a retryable error stopped, and rejects if text stays unwritten. */
  async function flush(): Promise<void> {
    for (const tab of tabs.values()) {
      if (isUnwritten(tab) && tab.error !== null && RETRYABLE_ERRORS.includes(tab.error)) {
        tab.state = 'unsaved'
        tab.error = null
      }
    }
    await Promise.all(Array.from(tabs.keys()).map((id) => writeTab(id).catch(() => markUnreachable(id))))
    const unwritten = Array.from(tabs.values()).filter(isUnwritten)
    if (unwritten.length > 0) {
      throw new Error(`The text of ${unwritten.map((tab) => tab.path).join(', ')} was not written`)
    }
  }

  function keep(id: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    tab.keepOverride = true
    tab.state = 'unsaved'
    writeTab(id).catch(() => markUnreachable(id))
  }

  function take(id: string): void {
    const tab = tabs.get(id)
    if (!tab) return
    notes
      .read(tab.path)
      .then((res) => {
        if (!tabs.has(id)) return
        if (res.ok) {
          tab.body = res.value.body
          tab.filePath = res.value.at ?? tab.path
          tab.baseline = { prose: res.value.body, stat: res.value.at ?? '' }
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
        notes
          .read(tab.path)
          .then((res) => {
            if (res.ok) {
              tab.body = res.value.body
              tab.baseline = { prose: res.value.body, stat: res.value.at ?? '' }
              tab.filePath = res.value.at ?? tab.filePath
              options?.onReplaced?.(tab.path)
            }
          })
          .catch(() => markUnreachable(tab.id))
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
    flushPending,
    flush,
    keep,
    take,
    stale,
    applyPathChanges,
  }
}
