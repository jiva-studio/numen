/**
 * Editing, conflict handling, and flushing for open notes and flashcard stores.
 */
import { cards } from '@/entities/deck/cards'
import { presets } from '@/entities/deck/presets'
import { useDeckTabs } from '@/widgets/deck-editor/composables/useDeckTabs'
import { useStencilTabs } from '@/widgets/stencil-editor/composables/useStencilTabs'
import { usePresetTab } from '@/widgets/preset-editor/composables/usePresetTab'
import { noteChanges } from '@/widgets/note-editor/changes'
import { openNotes } from '@/entities/note'
import { noteCreator } from '@/widgets/note-editor/maker'
import { useNoteTab } from '@/widgets/note-editor/composables/useNoteTab'
import { useFileFlush } from '@/features/file-conflict/flushing'
import { raiseConflicts } from '@/features/file-conflict/conflicts'
import { reaching, type Store } from '@/features/command-palette/deps'
import type { Core } from '@/app/ports/core'
import type { MessageLog } from '@/shared/notices/messages'
import type { FileOpeners } from '@/entities/tab/openers'
import type { useWindowTabs } from '@/entities/tab/windowTabs'

export interface NoteEditorsDeps {
  core: Core
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
  const reached = reaching(stores, puts)
  const titled = (id: string): string => stores.find((one) => one.has(id))?.called(id) ?? ''

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
    titled,
    kinds,
    close,
  }
}
