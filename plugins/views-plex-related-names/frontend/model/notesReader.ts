/** The notes related to the notes the plex shows, read once for every view that shows a note. */
import { computed, shallowRef, watch, type Ref, type ShallowRef } from 'vue'
import type {
  Caught,
  NoteGraph,
  PluginEvents,
  VaultChangeEvent,
  VaultNotes,
} from '@numen/plugin-sdk'
import type { NamesSettings } from './settings'

/** What the notes are read with and kept up to date through. */
export interface NotesReaderSource {
  readonly notes: Pick<VaultNotes, 'listRelatedNotes'>
  readonly events: Pick<PluginEvents, 'onVaultChange'>
  readonly settings: Pick<NamesSettings, 'showNames'>
  readonly reportFailure: (error: Caught) => void
}

/** The use of one note by one view: the notes related to it, `null` until they are read, while they are hidden and after a failed read, whether the first read has ended (answered or failed, and also when the names are hidden or the path is empty; it never goes back to `false`), and the end of the use. */
export interface NoteLease {
  readonly noteGraph: Readonly<Ref<NoteGraph | null>>
  readonly isLoaded: Readonly<Ref<boolean>>
  /** Ends the use. A note that no lease uses is forgotten. */
  release(): void
}

export interface NotesReader {
  /** Starts the use of the note at `path`; the views that use the same note share one read. */
  acquire(path: string): NoteLease
  /** Stops following the vault. */
  dispose(): void
}

/** How long a burst of changes of the vault waits before the notes are read once. */
const REFRESH_DELAY_MS = 500

/** The most notes the vault reads the related notes of in one call. */
const MAX_PATHS_PER_CALL = 64

interface Entry {
  readonly path: string
  readonly noteGraph: ShallowRef<NoteGraph | null>
  readonly isReadEnded: ShallowRef<boolean>
  leases: number
  latest: number
  isReading: boolean
}

export function createNotesReader(source: NotesReaderSource): NotesReader {
  const entries = new Map<string, Entry>()
  const changed = new Set<string>()
  let pending: ReturnType<typeof setTimeout> | undefined

  const queued = new Set<Entry>()
  let isFlushScheduled = false

  const readChunk = async (chunk: readonly Entry[]) => {
    const marks = chunk.map((entry) => ++entry.latest)
    const isCurrent = (entry: Entry, index: number) => marks[index] === entry.latest
    try {
      const answers = await source.notes.listRelatedNotes(
        chunk.map((entry) => entry.path),
        1,
      )
      chunk.forEach((entry, index) => {
        if (!isCurrent(entry, index)) return
        const answer = answers[index]
        entry.noteGraph.value = answer?.focus.path ? answer : null
        entry.isReading = false
        entry.isReadEnded.value = true
      })
    } catch (error) {
      let isReported = false
      chunk.forEach((entry, index) => {
        if (!isCurrent(entry, index)) return
        entry.noteGraph.value = null
        entry.isReading = false
        entry.isReadEnded.value = true
        if (!isReported) source.reportFailure(error)
        isReported = true
      })
    }
  }

  const flush = () => {
    isFlushScheduled = false
    const toRead = [...queued].filter((entry) => entries.get(entry.path) === entry)
    queued.clear()
    if (!source.settings.showNames.value) {
      for (const entry of toRead) entry.isReading = false
      return
    }
    for (let start = 0; start < toRead.length; start += MAX_PATHS_PER_CALL) {
      readChunk(toRead.slice(start, start + MAX_PATHS_PER_CALL)).catch(source.reportFailure)
    }
  }

  const read = (entry: Entry) => {
    entry.isReading = true
    queued.add(entry)
    if (isFlushScheduled) return
    isFlushScheduled = true
    queueMicrotask(flush)
  }

  const isAffected = (entry: Entry, event: VaultChangeEvent) =>
    !event.paths ||
    event.paths.length === 0 ||
    event.paths.some(
      (path) =>
        path === entry.path || entry.noteGraph.value?.related.some((item) => item.path === path),
    )

  const subscription = source.events.onVaultChange((event) => {
    if (!source.settings.showNames.value) return
    for (const entry of entries.values()) {
      if (isAffected(entry, event)) changed.add(entry.path)
    }
    if (changed.size === 0) return
    if (pending !== undefined) clearTimeout(pending)
    pending = setTimeout(() => {
      pending = undefined
      const paths = [...changed]
      changed.clear()
      if (!source.settings.showNames.value) return
      for (const path of paths) {
        const entry = entries.get(path)
        if (entry) read(entry)
      }
    }, REFRESH_DELAY_MS)
  })

  const stopWatching = watch(source.settings.showNames, (isShown) => {
    for (const entry of entries.values()) {
      if (isShown) {
        read(entry)
        continue
      }
      entry.latest += 1
      entry.isReading = false
      entry.noteGraph.value = null
    }
  })

  return {
    acquire: (path) => {
      if (!path) {
        return {
          noteGraph: shallowRef<NoteGraph | null>(null),
          isLoaded: shallowRef(true),
          release: () => undefined,
        }
      }
      let entry = entries.get(path)
      if (!entry) {
        entry = {
          path,
          noteGraph: shallowRef(null),
          isReadEnded: shallowRef(false),
          leases: 0,
          latest: 0,
          isReading: false,
        }
        entries.set(path, entry)
      }
      entry.leases += 1
      if (source.settings.showNames.value && !entry.isReading && !entry.noteGraph.value) {
        read(entry)
      }
      const held = entry
      const isLoaded = computed(() => !source.settings.showNames.value || held.isReadEnded.value)
      let isHeld = true
      return {
        noteGraph: held.noteGraph,
        isLoaded,
        release: () => {
          if (!isHeld) return
          isHeld = false
          held.leases -= 1
          if (held.leases > 0) return
          entries.delete(held.path)
          changed.delete(held.path)
          queued.delete(held)
        },
      }
    },
    dispose: () => {
      if (pending !== undefined) clearTimeout(pending)
      pending = undefined
      changed.clear()
      queued.clear()
      stopWatching()
      subscription.dispose()
    },
  }
}
