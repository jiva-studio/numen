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
  core: NotePort & Pick<VaultPort, 'watchQuit' | 'reportFlush'>
  log: MessageLog
  tabOpeners: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  day: () => string
}

export function useNoteEditors({ core, log, tabOpeners, held, day }: NoteEditorsDeps) {
  const changes = noteChanges()
  const notes = openNotes(core, { onReplaced: changes.handleNoteChange })
  const making = noteCreator(core, log.getWriter('made'))

  const noted = useNoteTab(core, notes, changes, held.handle, tabOpeners)
  const decks = useDeckTabs(cards, presets, held.handle, tabOpeners)
  const stencils = useStencilTabs(cards, held.handle, tabOpeners, log.getWriter('stencil'))
  const schedules = usePresetTab(
    presets,
    held.handle,
    tabOpeners,
    log.getWriter('preset'),
    day,
  )

  const fileFlush = useFileFlush(core)
  fileFlush.addHandler(notes.flush)
  fileFlush.addHandler(decks.flush)
  fileFlush.addHandler(stencils.flush)
  fileFlush.addHandler(schedules.flush)

  raiseConflicts(notes, fileFlush)
  raiseConflicts(decks, fileFlush)
  raiseConflicts(stencils, fileFlush)

  const stores: readonly Store[] = [noted.kept, decks.kept, stencils.kept]
  const reached = createNotes(stores, tabOpeners)
  const getTitle = (id: string): string => stores.find((one) => one.has(id))?.getTitle(id) ?? ''

  const kinds = [noted.kind, decks.kind, stencils.kind, schedules.kind]

  const close = () => {
    changes.close()
    fileFlush.close()
  }

  return {
    changes,
    notes,
    making,
    noted,
    decks,
    stencils,
    schedules,
    fileFlush,
    stores,
    reached,
    getTitle,
    kinds,
    close,
  }
}
