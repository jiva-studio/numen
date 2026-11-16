/** The names one side of a node draws: the notes related to its note, read while the names are shown. */
import { computed, shallowRef, watch, type ComputedRef } from 'vue'
import type { Caught, CommandsHost, NoteGraph, VaultNotes } from '@numen/plugin-sdk'
import { listNames } from '../lib/names'
import type { NamesSide, RelatedName } from '../types'
import type { NamesSettings } from './settings'

/** The command of the window that shows a note in a tab kind. */
const REVEAL = 'app:file.reveal'
const REVEAL_IN_PLEX = { kind: 'plex' }

/** What the names are read with and chosen through. */
export interface NamesSource {
  readonly notes: Pick<VaultNotes, 'listRelatedNotes'>
  readonly commands: Pick<CommandsHost, 'run'>
  readonly settings: NamesSettings
  readonly reportFailure: (error: Caught) => void
}

/**
 * The names `side` of the note at `path`, and `choose`, which shows the note of a name in the plex.
 * The notes related to the related notes are read too, with depth 2, and no name is made of them.
 */
export function useNames(
  source: NamesSource,
  side: NamesSide,
  path: () => string,
): { names: ComputedRef<readonly RelatedName[]>; choose: (path: string) => void } {
  const noteGraph = shallowRef<NoteGraph | null>(null)
  let latest = 0

  const read = async (notePath: string) => {
    latest += 1
    const mine = latest
    if (!notePath) {
      noteGraph.value = null
      return
    }
    try {
      const answer = await source.notes.listRelatedNotes(notePath, 2)
      if (mine === latest) noteGraph.value = answer
    } catch (error) {
      if (mine !== latest) return
      noteGraph.value = null
      source.reportFailure(error)
    }
  }

  watch(() => (source.settings.showNames.value ? path() : ''), read, { immediate: true })

  const names = computed(() =>
    noteGraph.value ? listNames(noteGraph.value, side, source.settings.namesPerNote.value) : [],
  )

  const choose = (notePath: string) => {
    const target = { path: notePath, title: '', isFolder: false, others: [], tab: '' }
    source.commands.run(REVEAL, target, undefined, REVEAL_IN_PLEX).catch(source.reportFailure)
  }

  return { names, choose }
}
