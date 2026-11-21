/** The names one side of a node draws: the notes related to its note, read while the names are shown. */
import { computed, onScopeDispose, shallowRef, watch, type ComputedRef } from 'vue'
import type { Caught, CommandsHost } from '@numen/plugin-sdk'
import { listNames } from '../lib/names'
import type { NamesSide, RelatedName } from '../types'
import type { NoteLease, NotesReader } from './notesReader'
import type { NamesSettings } from './settings'
import type { ShownPaths } from './shownPaths'

/** The command of the window that shows a note in a tab kind. */
const REVEAL = 'app:file.reveal'
const REVEAL_IN_PLEX = { kind: 'plex' }

/** What the names are read with and chosen through. */
export interface NamesSource {
  readonly reader: Pick<NotesReader, 'acquire'>
  readonly commands: Pick<CommandsHost, 'run'>
  readonly settings: NamesSettings
  readonly shown: ShownPaths
  readonly reportFailure: (error: Caught) => void
}

/** The names `side` of the note at `path`, whether their first read has ended, and `choose`, which shows the note of a name in the plex. */
export function useNames(
  source: NamesSource,
  side: NamesSide,
  path: () => string,
): {
  names: ComputedRef<readonly RelatedName[]>
  isLoaded: ComputedRef<boolean>
  choose: (path: string) => void
} {
  const lease = shallowRef<NoteLease>(source.reader.acquire(path()))
  watch(path, (next) => {
    lease.value.release()
    lease.value = source.reader.acquire(next)
  })
  onScopeDispose(() => lease.value.release())

  let unshow = source.shown.add(path())
  watch(path, (next) => {
    unshow()
    unshow = source.shown.add(next)
  })
  onScopeDispose(() => unshow())

  const names = computed(() => {
    const noteGraph = lease.value.noteGraph.value
    return noteGraph
      ? listNames(noteGraph, side, source.settings.namesPerNote.value, source.shown.has)
      : []
  })

  const isLoaded = computed(() => lease.value.isLoaded.value)

  const choose = (notePath: string) => {
    const target = { path: notePath, title: '', isFolder: false, others: [], tab: '' }
    source.commands.run(REVEAL, target, undefined, REVEAL_IN_PLEX).catch(source.reportFailure)
  }

  return { names, isLoaded, choose }
}
