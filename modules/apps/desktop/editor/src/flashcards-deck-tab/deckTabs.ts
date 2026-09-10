/**
 * The decks the window has open, and what one deck tab holds.
 *
 * A deck is saved the way a note is: one store for the whole window, the same
 * interval, and the same two answers where the file moved past what was read.
 * What travels between the store and the vault is the cards of the deck, and
 * the string the store is dirty against is those cards written out.
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

/** What one deck tab holds. */
export interface DeckTabState {
  /** The identity this deck opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The deck as the window draws it: the state it is in, and what it stands at. */
  readonly shown: ComputedRef<OpenNote>
  /** The cards, as the window holds them. */
  readonly deck: ComputedRef<BufferDeck>
  /** The same, as the grid draws them, each under the stencil that cuts it. */
  readonly drawn: ComputedRef<readonly DeckCard[]>
  /** The sections, as the grid draws them. */
  readonly sections: ComputedRef<readonly DeckSection[]>
  /** The stencils a card may be cut by. */
  readonly stencils: ComputedRef<readonly Stencil[]>
  /** What is wrong with the file, against the card it stands on. */
  readonly marks: ComputedRef<Marks>
  /** What the whole file was refused for, in words a person reads. */
  readonly saying: ComputedRef<string>
  /** The preset this deck is scheduled by. */
  readonly scheduled: ComputedRef<DeckPreset>
  /** The presets this deck may be put on, the defaults first. */
  readonly choices: ComputedRef<readonly Choice[]>
  /**
   * This deck put on the preset at that path, and on the defaults where the
   * path is empty. What it is owed reaches the file first.
   */
  schedules(preset: string): void
  /**
   * A card cut by that stencil, made at the end of the section named, and at
   * the end of the cards before the first section where none is.
   */
  adds(
    stencil: string,
    values: readonly { field: string; text: string }[],
    section: string | null,
  ): void
  removes(card: string): void
  /**
   * A card let go before another card, at the head of a section, or at the end
   * of the deck.
   */
  moves(card: string, at: string | null): void
  /**
   * One value of one card as it now reads. A card writing a field twice is
   * written where `nth` counts off under it.
   */
  writes(card: string, field: string, nth: number, text: string): void
  /** A section made at the end of the deck, under that name. */
  addsSection(name: string): void
  namesSection(section: string, name: string): void
  /** A section asked to go. Its heading goes, and the cards under it stay. */
  removesSection(section: string): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(id: string): void
}

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
        seen?.at ?? null,
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
      adds: (stencil, values, section) =>
        turns(id, added(deckAt(id), stencil, pathOfCut(offers.value, stencil), values, section)),
      removes: (card) => turns(id, removed(deckAt(id), card)),
      moves: (card, at) => turns(id, dropped(deckAt(id), card, at)),
      writes: (card, field, nth, text) => turns(id, filled(deckAt(id), card, field, nth, text)),
      addsSection: (name) => turns(id, sectionAdded(deckAt(id), name)),
      namesSection: (section, name) => turns(id, sectionNamed(deckAt(id), section, name)),
      removesSection: (section) => turns(id, sectionGone(deckAt(id), section)),
      keep: () => store.keep(id),
      take: () => store.take(id),
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
    asking: (id) => store.overtaken(id) !== null,
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
