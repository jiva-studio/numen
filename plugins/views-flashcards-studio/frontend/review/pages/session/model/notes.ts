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
import type { TextLink } from '@numen/plugin-sdk'
import type { LinkPiece } from '@numen/ui'

import { WORDS as words } from '../lib/notesWords'
import type { DeckNeighbourhood, Neighbour } from '../api/notes'

/** What the panel asks of the window it is drawn in. */
export interface NotesPanelDeps {
  readonly open: () => boolean
  /**
   * The panel asked for, or put away. The window can only be showing one thing,
   * so it is the window that holds which, and every panel moves that one thing.
   */
  readonly showPanel: (open: boolean) => void
  /** The vault the session is on. */
  readonly vault: () => string
  readonly deck: () => string
  /** What the deck is joined to. */
  readonly getDeckNeighbourhood: (vault: string, deck: string) => Promise<DeckNeighbourhood>
  /** Where the window says what a person has to know. */
  readonly showNotice: (message: string) => void
  /** The wikilinks of a text, as the core reads them. */
  readonly resolveLinks: (text: string) => Promise<readonly TextLink[]>
}

/** The links of one note's body, as the panel draws them. */
export interface NoteLinks {
  readonly links: readonly LinkPiece[]
  /** The addresses that reach no note. */
  readonly unresolved: readonly string[]
}

/** A note is keyed by what its article is keyed by. */
export const getNoteKey = (one: Neighbour): string => one.path || one.written

const getNoteLinks = (found: readonly TextLink[]): NoteLinks => ({
  links: found.map((one) => ({ raw: one.raw, href: one.address, text: one.text })),
  unresolved: [...new Set(found.filter((one) => one.path === '').map((one) => one.address))],
})

function useDeckNotes(
  getDeckNeighbourhood: (vault: string, deck: string) => Promise<DeckNeighbourhood>,
  showNotice: (message: string) => void,
  resolveLinks: (text: string) => Promise<readonly TextLink[]>,
) {
  const notes = shallowRef<readonly Neighbour[]>([])
  const links = shallowRef<ReadonlyMap<string, NoteLinks>>(new Map())
  const unread = ref(0)
  const isWorking = ref(false)
  let activeDeck: string | null = null
  let pendingDeck: string | null = null

  const clearNotes = () => {
    notes.value = []
    links.value = new Map()
    unread.value = 0
    activeDeck = null
    pendingDeck = null
  }

  const fetchLinks = async (deck: string, one: Neighbour) => {
    let found: readonly TextLink[]
    try {
      found = await resolveLinks(one.body)
    } catch {
      // A body whose links cannot be read is drawn with its links as text.
      return
    }
    if (activeDeck !== deck || !notes.value.includes(one)) return
    links.value = new Map(links.value).set(getNoteKey(one), getNoteLinks(found))
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
      for (const one of neighbourhood.notes) {
        if (one.body) void fetchLinks(deck, one)
      }
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

  return { notes, links, unread, isWorking, fetchNotes, clearNotes }
}

export function useNotesPanel(deps: NotesPanelDeps) {
  const open = computed(() => deps.open())
  const activePath = ref('')
  const { notes, links, unread, isWorking, fetchNotes, clearNotes } = useDeckNotes(
    deps.getDeckNeighbourhood,
    deps.showNotice,
    deps.resolveLinks,
  )

  const openPanel = async (note = '') => {
    const deck = deps.deck()
    if (!deck) return
    activePath.value = note
    deps.showPanel(true)
    await fetchNotes(deps.vault(), deck)
  }

  const closePanel = () => {
    deps.showPanel(false)
  }

  const readItem = () => {
    activePath.value = ''
  }

  const endSession = () => {
    clearNotes()
    activePath.value = ''
    deps.showPanel(false)
  }

  const api = {
    notes,
    links,
    unread,
    isWorking,
    col: activePath,
    open,
    openPanel,
    closePanel,
    read: readItem,
    endSession,
  }
  return api
}

/** What one panel holds. */
export type NotesPanelState = ReturnType<typeof useNotesPanel>
