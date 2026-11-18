/** The notes related to the notes the plex shows, read once for every view that shows a note. */
import { shallowRef, type Ref, type ShallowRef } from 'vue'
import type { Caught, NoteGraph, PluginEvents, VaultNotes } from '@numen/plugin-sdk'

/** What the notes are read with and kept up to date through. */
export interface NotesReaderSource {
  readonly notes: Pick<VaultNotes, 'listRelatedNotes'>
  readonly events: Pick<PluginEvents, 'onVaultChange'>
  readonly reportFailure: (error: Caught) => void
}

/** The use of one note by one view: the notes related to it, `null` until they are read and after a failed read, whether the first read has ended (answered or failed, and also when the path is empty; it never goes back to `false`), and the end of the use. */
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

  const subscription = source.events.onVaultChange(() => {
    for (const entry of entries.values()) {
      changed.add(entry.path)
    }
    if (changed.size === 0) return
    if (pending !== undefined) clearTimeout(pending)
    pending = setTimeout(() => {
      pending = undefined
      const paths = [...changed]
      changed.clear()
      for (const path of paths) {
        const entry = entries.get(path)
        if (entry) read(entry)
      }
    }, REFRESH_DELAY_MS)
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
      if (!entry.isReading && !entry.noteGraph.value) read(entry)
      const leased = entry
      let isActive = true
      return {
        noteGraph: leased.noteGraph,
        isLoaded: leased.isReadEnded,
        release: () => {
          if (!isActive) return
          isActive = false
          leased.leases -= 1
          if (leased.leases > 0) return
          entries.delete(leased.path)
          changed.delete(leased.path)
          queued.delete(leased)
        },
      }
    },
    dispose: () => {
      if (pending !== undefined) clearTimeout(pending)
      pending = undefined
      changed.clear()
      queued.clear()
      subscription.dispose()
    },
  }
}
