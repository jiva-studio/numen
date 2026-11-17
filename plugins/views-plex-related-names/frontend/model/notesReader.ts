/** The notes related to the notes the plex shows, read once for every view that shows a note. */
import { shallowRef, watch, type Ref, type ShallowRef } from 'vue'
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

/** The use of one note by one view: the notes related to it, `null` until they are read, while they are hidden and after a failed read, and the end of the use. */
export interface NoteLease {
  readonly noteGraph: Readonly<Ref<NoteGraph | null>>
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

interface Entry {
  readonly path: string
  readonly noteGraph: ShallowRef<NoteGraph | null>
  leases: number
  latest: number
  isReading: boolean
}

export function createNotesReader(source: NotesReaderSource): NotesReader {
  const entries = new Map<string, Entry>()
  const changed = new Set<string>()
  let pending: ReturnType<typeof setTimeout> | undefined

  const read = async (entry: Entry) => {
    entry.latest += 1
    const mine = entry.latest
    entry.isReading = true
    try {
      const [answer] = await source.notes.listRelatedNotes([entry.path], 1)
      if (mine !== entry.latest) return
      entry.noteGraph.value = answer ?? null
    } catch (error) {
      if (mine !== entry.latest) return
      entry.noteGraph.value = null
      source.reportFailure(error)
    }
    entry.isReading = false
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
        if (entry) read(entry).catch(source.reportFailure)
      }
    }, REFRESH_DELAY_MS)
  })

  const stopWatching = watch(source.settings.showNames, (isShown) => {
    for (const entry of entries.values()) {
      if (isShown) {
        read(entry).catch(source.reportFailure)
        continue
      }
      entry.latest += 1
      entry.isReading = false
      entry.noteGraph.value = null
    }
  })

  return {
    acquire: (path) => {
      if (!path) return { noteGraph: shallowRef<NoteGraph | null>(null), release: () => undefined }
      let entry = entries.get(path)
      if (!entry) {
        entry = { path, noteGraph: shallowRef(null), leases: 0, latest: 0, isReading: false }
        entries.set(path, entry)
      }
      entry.leases += 1
      if (source.settings.showNames.value && !entry.isReading && !entry.noteGraph.value) {
        read(entry).catch(source.reportFailure)
      }
      const held = entry
      let isHeld = true
      return {
        noteGraph: held.noteGraph,
        release: () => {
          if (!isHeld) return
          isHeld = false
          held.leases -= 1
          if (held.leases > 0) return
          entries.delete(held.path)
          changed.delete(held.path)
        },
      }
    },
    dispose: () => {
      if (pending !== undefined) clearTimeout(pending)
      pending = undefined
      changed.clear()
      stopWatching()
      subscription.dispose()
    },
  }
}
