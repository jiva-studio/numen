/**
 * Editing, conflict handling, and flushing for open notes and flashcard stores.
 */
import { cards } from '../shared/flashcards/cards'
import { presets } from '../flashcards-preset-tab/core'
import { useDeckTabs } from '../flashcards-deck-tab/deckTabs'
import { useStencilTabs } from '../flashcards-stencil-tab/stencilTabs'
import { usePresetTab } from '../flashcards-preset-tab/kind'
import { noteChanges } from '../note-tab/changes'
import { openNotes } from '../note-tab/notes'
import { noteMaker } from '../note-tab/maker'
import { useNoteTab } from '../note-tab/kind'
import { useFileFlush } from '../shared/saving/flushing'
import { raiseConflicts } from '../shared/saving/conflicts'
import { reaching, type Store } from '../shared/command/deps'
import type { Core } from '../shared/core'
import type { MessageLog } from '../shared/notices/messages'
import type { FileOpeners } from '../shared/tabs/openers'
import type { useWindowTabs } from '../shared/tabs/windowTabs'

export interface EditingDeps {
  core: Core
  log: MessageLog
  puts: FileOpeners
  held: ReturnType<typeof useWindowTabs>
  day: () => string
}

export function openEditing({ core, log, puts, held, day }: EditingDeps) {
  const changes = noteChanges()
  const notes = openNotes(core, { replaced: changes.arrived })
  const making = noteMaker(core, log.under('made'))

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
