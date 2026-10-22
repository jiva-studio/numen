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
  readonly getDeckNeighbourhood: (vault: string, deck: string) => Promise<DeckNeighbourhood>
  /** Where the window says what a person has to know. */
  readonly showNotice: (message: string) => void
}

function useDeckNotes(
  getDeckNeighbourhood: (vault: string, deck: string) => Promise<DeckNeighbourhood>,
  showNotice: (message: string) => void,
) {
  const notes = shallowRef<readonly Neighbour[]>([])
  const unread = ref(0)
  const isWorking = ref(false)
  let activeDeck: string | null = null
  let pendingDeck: string | null = null

  const clearNotes = () => {
    notes.value = []
    unread.value = 0
    activeDeck = null
    pendingDeck = null
  }

  const fetchNotes = async (vault: string, deck: string) => {
    if (activeDeck === deck) return
    clearNotes()
    pendingDeck = deck
    isWorking.value = true
    let failed = false
    try {
      const neighbourhood = await getDeckNeighbourhood(vault, deck)
      if (pendingDeck !== deck) return
      notes.value = neighbourhood.notes
      unread.value = neighbourhood.unread
      activeDeck = deck
    } catch {
      if (pendingDeck !== deck) return
      failed = true
      showNotice(words.unreached)
    } finally {
      if (pendingDeck === deck) {
        if (failed) clearNotes()
        isWorking.value = false
      }
    }
  }

  return { notes, unread, isWorking, fetchNotes, clearNotes }
}

export function useNotesPanel(deps: NotesPanelDeps) {
  const open = computed(() => deps.open())
  const at = ref('')
  const { notes, unread, isWorking, fetchNotes, clearNotes } = useDeckNotes(
    deps.getDeckNeighbourhood,
    deps.showNotice,
  )

  const openPanel = async (note = '') => {
    const deck = deps.deck()
    if (!deck) return
    at.value = note
    deps.showPanel(true)
    await fetchNotes(deps.vault(), deck)
  }

  const closePanel = () => {
    deps.showPanel(false)
  }

  const readItem = () => {
    at.value = ''
  }

  const endSession = () => {
    clearNotes()
    at.value = ''
    deps.showPanel(false)
  }

  return {
    notes,
    unread,
    isWorking,
    at,
    open,
    openPanel,
    closePanel,
    read: readItem,
    endSession,
  }
}

/** What one panel holds. */
export type NotesPanelState = ReturnType<typeof useNotesPanel>
