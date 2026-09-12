/**
 * The panel the deck's notes are read in: what is in it, and what it is asked
 * for.
 *
 * Everything the panel is, is here. What draws it reads this and decides
 * nothing.
 *
 * What is read belongs to the deck and not to the card, because a link is
 * written in a file and the file is the deck. So it is asked for once a deck
 * and kept while the cards of that deck go by; a session over a whole vault
 * walks several decks, and each is asked for as it comes round.
 */
import { computed, ref, shallowRef } from 'vue'

import { WORDS as words } from '../lib/notesWords'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'

/** What the panel asks of the window it is drawn in. */
export interface NotesPanelDeps {
  /** Whether the panel is what the window is showing. */
  readonly open: () => boolean
  /**
   * The panel asked for, or put away. The window can only be showing one thing,
   * so it is the window that holds which, and every panel moves that one thing.
   */
  readonly showPanel: (open: boolean) => void
  /** The vault the session is on. */
  readonly vault: () => string
  /** The deck the card in front of the person stands in, empty between cards. */
  readonly deck: () => string
  /** What the deck is joined to. */
  readonly around: (vault: string, deck: string) => Promise<DeckNeighbourhood>
  /** Where the window says what a person has to know. */
  readonly showNotice: (said: string) => void
}

export function useNotesPanel(deps: NotesPanelDeps) {
  /** Whether the panel is what the window is showing. */
  const open = computed(() => deps.open())

  const notes = shallowRef<readonly Neighbour[]>([])

  /** How many at the end came named and not read. */
  const unread = ref(0)

  const working = ref(false)

  /** The deck the notes in hand belong to, empty while none are. */
  const held = ref('')

  /** The note the panel is to be opened on, where a link named one. */
  const at = ref('')

  /** The deck the answer being waited for was asked about. */
  let asked = ''

  /** What the deck is joined to, asked for once and kept until the deck changes. */
  const fetchNotes = async (vault: string, deck: string) => {
    if (held.value === deck) return
    // What is in hand belongs to the deck behind this one, and a person must
    // never read one deck's notes under another deck's card.
    clearNotes()
    asked = deck
    working.value = true
    try {
      const around = await deps.around(vault, deck)
      // A card answered while this was in flight moves the session to another
      // deck, and what came back is then about the deck behind it.
      if (asked !== deck) return
      notes.value = around.notes
      unread.value = around.unread
      held.value = deck
    } catch {
      // The deck could not be reached, and the window says so: the panel holds
      // nothing until it is asked again.
      if (asked !== deck) return
      notes.value = []
      unread.value = 0
      held.value = ''
      deps.showNotice(words.unreached)
    } finally {
      if (asked === deck) working.value = false
    }
  }

  /**
   * The panel asked for, on whichever deck is up. Named is the note it is
   * opened on, where a link in the card named one.
   */
  const openPanel = async (named = '') => {
    const deck = deps.deck()
    if (!deck) return
    at.value = named
    deps.showPanel(true)
    await fetchNotes(deps.vault(), deck)
  }

  /** The panel put away, with what was read in it kept. */
  const closePanel = () => {
    deps.showPanel(false)
  }

  /** The note the panel was opened on has been read to, and is not sought again. */
  const read = () => {
    at.value = ''
  }

  const clearNotes = () => {
    notes.value = []
    unread.value = 0
    held.value = ''
    asked = ''
  }

  /** The session is over: the panel holds nothing and is put away. */
  const endSession = () => {
    clearNotes()
    at.value = ''
    deps.showPanel(false)
  }

  return {
    notes,
    unread,
    working,
    at,
    open,
    openPanel,
    closePanel,
    read,
    endSession,
  }
}

/** What one panel holds. */
export type NotesPanelState = ReturnType<typeof useNotesPanel>
