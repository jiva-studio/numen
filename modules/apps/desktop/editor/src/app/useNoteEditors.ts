/**
 * Editing, conflict handling, and flushing for open notes and flashcard stores.
 */
import { cards } from '@/entities/deck'
import { presets } from '@/entities/deck'
import { useDeckTabs } from '@/pages/deck-editor'
import { useStencilTabs } from '@/pages/stencil-editor'
import { usePresetTab } from '@/pages/preset-editor'
import { noteChanges, noteCreator, useNoteTab } from '@/pages/note-editor'
import { openNotes } from '@/entities/note'
import { raiseConflicts, useFileFlush } from '@/features/file-conflict'
import { createNotes, type Store } from '@/features/command-palette'
import type { NotePort } from '@/app/ports/notes'
import type { VaultPort } from '@/app/ports/vault'
import type { MessageLog } from '@/shared/notices/messages'
import type { FileOpeners } from '@/entities/tab'
import type { useWindowTabs } from '@/entities/tab'

export interface NoteEditorsDeps {
  core: NotePort & Pick<VaultPort, 'quitting' | 'flushed'>
  log: MessageLog
  puts: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  day: () => string
}

export function useNoteEditors({ core, log, puts, held, day }: NoteEditorsDeps) {
  const changes = noteChanges()
  const notes = openNotes(core, { replaced: changes.arrived })
  const making = noteCreator(core, log.under('made'))

  const noted = useNoteTab(core, notes, changes, held.handle, puts)
  const decks = useDeckTabs(cards, presets, held.handle, puts)
  const stencils = useStencilTabs(cards, held.handle, puts, log.under('stencil'))
  const schedules = usePresetTab(
    presets,
    held.handle,
    puts,
    log.under('preset'),
    day,
  )

  const going = useFileFlush(core)
  going.holds(notes.flush)
  going.holds(decks.flush)
  going.holds(stencils.flush)
  going.holds(schedules.flush)

  raiseConflicts(notes, going)
  raiseConflicts(decks, going)
  raiseConflicts(stencils, going)

  const stores: readonly Store[] = [noted.kept, decks.kept, stencils.kept]
  const reached = createNotes(stores, puts)
  const getTitle = (id: string): string => stores.find((one) => one.has(id))?.called(id) ?? ''

  const kinds = [noted.kind, decks.kind, stencils.kind, schedules.kind]

  const close = () => {
    changes.close()
    going.close()
  }

  return {
    changes,
    notes,
    making,
    noted,
    decks,
    stencils,
    schedules,
    going,
    stores,
    reached,
    getTitle,
    kinds,
    close,
  }
}
