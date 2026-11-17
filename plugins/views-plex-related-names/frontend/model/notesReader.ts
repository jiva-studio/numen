/** The notes related to the notes the plex shows, read once for every view that shows a note. */
import { shallowRef, type Ref } from 'vue'
import type { Caught, NoteGraph, PluginEvents, VaultNotes } from '@numen/plugin-sdk'
import type { NamesSettings } from './settings'

/** What the notes are read with and kept up to date through. */
export interface NotesReaderSource {
  readonly notes: Pick<VaultNotes, 'listRelatedNotes'>
  readonly events: Pick<PluginEvents, 'onVaultChange'>
  readonly settings: Pick<NamesSettings, 'showNames'>
  readonly reportFailure: (error: Caught) => void
}

/** The use of one note by one view: the answer to read, and the end of the use. */
export interface NoteLease {
  /** The notes related to the note, `null` until they are read, while they are hidden and after a failed read. */
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

export function createNotesReader(_source: NotesReaderSource): NotesReader {
  return {
    acquire: () => ({ noteGraph: shallowRef<NoteGraph | null>(null), release: () => undefined }),
    dispose: () => undefined,
  }
}
