/**
 * Window registration and tab state for flashcard deck tabs.
 */
import { computed, shallowRef, type ComputedRef } from 'vue'
import type { DeckCard, DeckSection, PlexShowing, Stencil } from '@numen/ui'
import type { Move } from '../shared/core'
import type { Cards, StencilSummary } from '../shared/flashcards/cards'
import type { Store } from '../shared/command/deps'
import type { Presets } from '../flashcards-preset-tab/core'
import { answers } from './answers'
import { reader } from './reader'
import { useDeckSchedule, type Choice, type DeckPreset } from './scheduler'
import { openNotes, type OpenNote } from '../note-tab/notes'
import { markOf } from '../note-tab/tab'
import type { TabKind, WindowHandle } from '../shared/tabs/windowTabs'
import type { FileOpeners } from '../shared/tabs/openers'
import { DECK } from '../shared/tabs/workspace'
import DeckTab from './DeckTab.vue'
import {
  added,
  drawnSectionsOf,
  deckBodyOf,
  cardsOf,
  dropped,
  stencilsOf,
  deckIn,
  deckOf,
  drawnOf,
  filled,
  pathOfCut,
  removed,
  sameOffers,
  sectionAdded,
  sectionGone,
  sectionNamed,
  sectionsOf,
  type BufferDeck,
} from './deck'
import type { Marks } from '../shared/flashcards/marks'
import type { DeckTabState } from './types'

export type { DeckTabState }

export function useDeckTabs(cards: Cards, presets: Presets, handle: WindowHandle, puts: FileOpeners) {
  /** What the vault last said about each file this window holds. */
  const said = answers()
  /** The stencils of the vault, as they were last listed. */
  const offers = shallowRef<readonly StencilSummary[]>([])
  /** Whether the last listing of the stencils answered. */
  let listedOk = true

  const store = openNotes({
    read: async (path) => {
      const answer = await cards.readDeck(path)
      const deck = answer.deck ? deckOf(answer.deck) : null
      said.reads(path, {
        problems: answer.deck?.problems ?? [],
        refusal: answer.refusal,
        bound: answer.bound,
        title: answer.deck?.title ?? null,
      })
      if (answer.refusal !== null) return { body: '', refusal: answer.refusal }
      return { body: deck ? deckBodyOf(deck) : '', refusal: null, at: answer.at }
    },
    write: async (path, body, seen) => {
      const deck = deckIn(body)
      const answer = await cards.writeDeck(
        path,
        {
          preamble: deck.preamble,
          cards: cardsOf(deck),
          sections: sectionsOf(deck),
          tail: deck.tail,
        },
        seen?.path ?? null,
      )
      said.writes(path, { refusal: answer.refusal, bound: answer.bound })
      return {
        body: '',
        refusal: answer.refusal,
        at: answer.at,
        changed: answer.changed,
      }
    },
  })

  const read = reader(store, said.problemsAt)
  const { deckAt, marksAt } = read

  /** The stencils a card may be cut by, made again where the list changed. */
  const stencils = computed(() => stencilsOf(offers.value))

  /** A deck as it now stands, written back into the store. */
  const turns = (id: string, deck: BufferDeck): void => {
    const body = deckBodyOf(deck)
    read.holds(id, body, deck)
    store.typed(id, body)
  }

  /**
   * The stencils of the vault, asked for again. A list naming the same
   * stencils leaves the one held standing, so what is drawn under it stands
   * with it.
   */
  const lists = async (): Promise<void> => {
    try {
      const listed = (await cards.stencils()).stencils
      listedOk = true
      if (!sameOffers(offers.value, listed)) offers.value = listed
    } catch {
      // The stencils the window last heard of stand, and a card is cut by one
      // of them until the vault answers again.
      listedOk = false
    }
  }

  /** The stencils asked for again, where the last listing did not answer. */
  const listsAgain = (): void => {
    if (!listedOk) void lists()
  }

  const scheduled = useDeckSchedule(presets, store)
  const { choices, listsPresets, listsPresetsAgain, asks, schedules, scheduledAt } = scheduled

  const held = (id: string): DeckTabState => {
    /** The deck this tab is showing, which everything drawn of it follows. */
    const deck = computed(() => deckAt(id))

    return {
      id,
      shown: computed(() => store.shown(id)),
      deck,
      drawn: computed(() => drawnOf(deck.value, offers.value)),
      sections: computed(() => drawnSectionsOf(deck.value)),
      stencils,
      marks: computed(() => marksAt(id)),
      saying: computed(() => said.saying(store.where(id), store.shown(id).refusal !== null)),
      scheduled: computed(() => scheduledAt(id)),
      choices,
      schedules: (preset) => void schedules(id, preset),
      setSchedule: (preset) => void schedules(id, preset),
      adds: (stencil, values, section) =>
        turns(id, added(deckAt(id), stencil, pathOfCut(offers.value, stencil), values, section)),
      addCard: (stencil, values, section) =>
        turns(id, added(deckAt(id), stencil, pathOfCut(offers.value, stencil), values, section)),
      removes: (card) => turns(id, removed(deckAt(id), card)),
      removeCard: (card) => turns(id, removed(deckAt(id), card)),
      moves: (card, at) => turns(id, dropped(deckAt(id), card, at)),
      moveCard: (card, at) => turns(id, dropped(deckAt(id), card, at)),
      writes: (card, field, nth, text) => turns(id, filled(deckAt(id), card, field, nth, text)),
      writeCardField: (card, field, nth, text) => turns(id, filled(deckAt(id), card, field, nth, text)),
      addsSection: (name) => turns(id, sectionAdded(deckAt(id), name)),
      addSection: (name) => turns(id, sectionAdded(deckAt(id), name)),
      namesSection: (section, name) => turns(id, sectionNamed(deckAt(id), section, name)),
      renameSection: (section, name) => turns(id, sectionNamed(deckAt(id), section, name)),
      removesSection: (section) => turns(id, sectionGone(deckAt(id), section)),
      removeSection: (section) => turns(id, sectionGone(deckAt(id), section)),
      keep: () => store.keep(id),
      keepMine: () => store.keep(id),
      take: () => store.take(id),
      takeFile: () => store.take(id),
      /** The tab stands until the deck says the write is done, and goes then. */
      shuts: (tab) => {
        const path = store.where(id)
        void store.shut(id).then((gone) => {
          if (!gone) return
          read.closes(id)
          scheduled.closes(id)
          forgets(path)
          handle.closes(tab)
        })
      },
      close: (tab) => {
        const path = store.where(id)
        void store.shut(id).then((gone) => {
          if (!gone) return
          read.closes(id)
          scheduled.closes(id)
          forgets(path)
          handle.closes(tab)
        })
      },
    }
  }

  /**
   * What the vault said about a file no tab of this window stands at any
   * longer. A second tab standing there keeps it.
   */
  const forgets = (path: string): void => {
    if (store.all().some((one) => store.where(one) === path)) return
    said.forgets(path)
    scheduled.forgets(path)
  }

  /** The tab holding a deck lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = handle.each<DeckTabState>(DECK).find((one) => one.state.id === id)
    tab?.state.shuts(tab.id)
  }

  /** The decks, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.all().includes(id),
    where: (id) => store.where(id),
    called: (id) => said.called(store.where(id)),
    asking: (id) => (store.stale?.(id) ?? store.overtaken(id)) !== null,
    settles: (id) => store.settles(id),
    shuts,
    holding: (path) => store.all().find((id) => store.where(id) === path) ?? null,
  }

  /**
   * Every open deck under the file it stands at now, against the identity it
   * opened under. A deck that moved is looked up here to reach the tab already
   * holding it.
   */
  const tabbed = computed<ReadonlyMap<string, string>>(
    () => new Map(store.all().map((one) => [store.where(one), one])),
  )

  /**
   * The identity minted for a deck asked for by name, until its tab opens under
   * it. A deck is asked for and shown in two steps, and both name the same tab.
   */
  const minting = new Map<string, string>()
  const minted = new Map<string, string>()

  /** The identity of the tab standing at a file, minted where none stands there. */
  const mints = (path: string): string => {
    const open = tabbed.value.get(path) ?? minting.get(path)
    if (open) return open
    const one = crypto.randomUUID()
    minting.set(path, one)
    minted.set(one, path)
    return one
  }

  /**
   * A deck tab as the window keeps it. A deck is its own tab, filed under the
   * identity it opened under, so the same file asked for twice is the tab it
   * has wherever the file has been renamed to since.
   */
  const kind: TabKind<DeckTabState, typeof DECK> = {
    kind: DECK,
    opens: (id) => {
      const path = minted.get(id) ?? id
      store.open(id, path)
      minting.delete(path)
      minted.delete(id)
      void lists()
      void listsPresets()
      void asks(path)
      return held(id)
    },
    called: (one) => said.called(store.where(one.id)),
    marked: (one) => markOf(one.shown.value.state),
    draws: DeckTab,
    identity: (id) => id,
    // A tab back on screen is a tab a person is about to draw cards in, so a
    // listing that never answered is asked for again. Both of them are.
    shown: () => {
      listsAgain()
      listsPresetsAgain()
    },
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    // What an open deck owes at the quit is written by the quit, which the
    // window waits for.
    gone: () => {},
  }

  /** A deck put in front of the person, in a tab of its own. */
  const shows = (path: string, title = '', showing: PlexShowing = 'here'): void => {
    const id = mints(path)
    if (title) said.names(path, title)
    void (showing === 'beside' ? handle.beside(DECK, id) : handle.opens(DECK, id))
  }

  // The editor of a deck, which is the grid of its cards. A card stands on no
  // line of prose, so a deck asked for at a place inside it opens whole.
  puts.holds('deck', shows)

  /**
   * The vault changed: every open deck hears it, and the stencils and the
   * presets are listed again. A window holding no deck asks for neither.
   */
  const changed = (paths: readonly string[], renamed: readonly Move[] = []): void => {
    // What the vault said about a file is filed under that file, so a file
    // that moved takes it along.
    for (const went of renamed) {
      said.moved(went.from, went.to)
      scheduled.moved(went.from, went.to)
    }
    store.changed(paths, renamed)
    if (store.all().length === 0) return
    void lists()
    // A preset made, removed or renamed changes what a deck may be put on, and
    // a deck written changes which preset it names.
    void listsPresets()
    for (const one of store.all()) void asks(store.where(one))
  }

  return {
    kind,
    held,
    changed,
    lists,
    called: said.called,
    kept,
    /** Every open deck, for the quit and for the question it raises. */
    all: store.all,
    shown: store.shown,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
