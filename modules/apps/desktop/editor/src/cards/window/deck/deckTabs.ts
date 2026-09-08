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
import type { Move, RefusalReason } from '../../../core'
import type { Cards, Problem, StencilSummary } from '../../vault/cards'
import type { Store } from '../../../command/deps'
import type { Presets } from '../../../preset/core'
import { fileOf } from '../../../paths'
import { reader } from './reader'
import { scheduler, type Choice, type DeckPreset } from './scheduler'
import { openNotes, type OpenNote } from '../../../note/notes'
import { markOf } from '../../../note/tab'
import type { Kind, WindowHandle } from '../../../tabs/windowing'
import type { FileOpeners } from '../../../tabs/openers'
import { DECK } from '../../../tabs/workspace'
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
  type Deck,
} from './deck'
import type { Marks } from '../marks'
import { WORDS as words } from '../words'

/** What the vault said about one file the last time it was read or written. */
interface VaultAnswer {
  /**
   * What is wrong with the file, in the order the cards were read in. Which
   * card each stands on is decided against the deck the grid is drawing, so a
   * card the window is holding through a re-read keeps its mark.
   */
  readonly problems: readonly Problem[]
  /** What the last read of the file was refused for. */
  readonly reading: RefusalReason | null
  /** What the last write of it was refused for. */
  readonly writing: RefusalReason | null
  /** The size a deck is read up to, where that is what refused it. */
  readonly bound: number
}

const NOTHING: VaultAnswer = { problems: [], reading: null, writing: null, bound: 0 }

/** What one deck tab holds. */
export interface DeckTabState {
  /** The identity this deck opened under, which its tab keeps wherever it goes. */
  readonly id: string
  /** The deck as the window draws it: the state it is in, and what it stands at. */
  readonly shown: ComputedRef<OpenNote>
  /** The cards, as the window holds them. */
  readonly deck: ComputedRef<Deck>
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

export function decking(cards: Cards, presets: Presets, handle: WindowHandle, puts: FileOpeners) {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, VaultAnswer>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()
  /** The stencils of the vault, as they were last listed. */
  const offers = shallowRef<readonly StencilSummary[]>([])
  /** Whether the last listing of the stencils answered. */
  let listedOk = true

  const store = openNotes({
    read: async (path) => {
      const answer = await cards.readDeck(path)
      const deck = answer.deck ? deckOf(answer.deck) : null
      told.set(path, {
        problems: answer.deck?.problems ?? [],
        reading: answer.refusal,
        writing: null,
        bound: answer.bound,
      })
      if (answer.deck) titles.set(path, answer.deck.title)
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
      const said = told.get(path) ?? NOTHING
      told.set(path, {
        problems: said.problems,
        reading: said.reading,
        writing: answer.refusal,
        bound: answer.bound,
      })
      return {
        body: '',
        refusal: answer.refusal,
        at: answer.at,
        changed: answer.changed,
      }
    },
  })

  const read = reader(store, (path) => (told.get(path) ?? NOTHING).problems)
  const { deckAt, marksAt } = read

  /** The stencils a card may be cut by, made again where the list changed. */
  const stencils = computed(() => stencilsOf(offers.value))

  /** A deck as it now stands, written back into the store. */
  const turns = (id: string, deck: Deck): void => {
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

  const scheduled = scheduler(presets, store)
  const { choices, listsPresets, listsPresetsAgain, asks, schedules, scheduledAt } = scheduled


  /** The words one refusal is put in, and nothing for one this file has none for. */
  const whyOf = (refusal: RefusalReason | null, bound: number): string | null => {
    if (refusal === 'deckTooLarge') return words.tooLarge(bound)
    if (refusal === 'notADeck') return words.notADeck
    return null
  }

  /**
   * What one tab was refused for, in words a person reads. A vault that
   * answered nothing at all left the tab refused and said no word of its own.
   */
  const sayingOf = (id: string): string => {
    if (store.shown(id).refusal === null) return ''
    const said = told.get(store.where(id)) ?? NOTHING
    if (said.reading !== null) return whyOf(said.reading, said.bound) ?? words.refused
    if (said.writing !== null) return whyOf(said.writing, said.bound) ?? words.notSaved
    return words.unreachable
  }

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
      saying: computed(() => sayingOf(id)),
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
    told.delete(path)
    titles.delete(path)
    scheduled.forgets(path)
  }

  /** What a deck tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => titles.get(path) || fileOf(path)

  /** The tab holding a deck lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = handle.each<DeckTabState>(DECK).find((one) => one.state.id === id)
    tab?.state.shuts(tab.id)
  }

  /** The decks, as a command reaches the ones the window has open. */
  const kept: Store = {
    has: (id) => store.all().includes(id),
    where: (id) => store.where(id),
    called: (id) => called(store.where(id)),
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
  const kind: Kind<DeckTabState> = {
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
    called: (one) => called(store.where(one.id)),
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
    if (title) titles.set(path, title)
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
      const said = told.get(went.from)
      if (said) told.set(went.to, said)
      told.delete(went.from)
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
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
    called,
    kept,
    /** Every open deck, for the quit and for the question it raises. */
    all: store.all,
    shown: store.shown,
    keep: store.keep,
    take: store.take,
    flush: store.flush,
  }
}
