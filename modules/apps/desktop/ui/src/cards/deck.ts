/**
 * The decks the window has open, and what one deck tab holds.
 *
 * A deck is saved the way a note is: one store for the whole window, the same
 * interval, and the same two answers where the file moved past what was read.
 * What travels between the store and the vault is the cards of the deck, and
 * the string the store is dirty against is those cards written out.
 */
import { shallowRef } from 'vue'
import type { Cut, Drawn } from '@numen/ui'
import type { Cards, Offer, Refused, Went } from '../core'
import type { Store } from '../doing'
import { editing, type Editing } from '../note/editing'
import { markOf } from '../note/tab'
import type { Host, Kind } from '../windowing'
import { DECK } from '../workspace'
import DeckTab from './DeckTab.vue'
import {
  added,
  bodyOf,
  cardsOf,
  carried,
  cutsOf,
  deckIn,
  deckOf,
  drawnOf,
  filled,
  marksOf,
  named,
  names,
  pathOfCut,
  removed,
  type Deck,
  type Marks,
} from './model'
import { WORDS as words } from './words'

/** What the vault said about one file the last time it was read or written. */
interface Told {
  /**
   * What is wrong, against the card it was read against. A problem stands on
   * the card it came in on, so a card carried elsewhere in the order or a card
   * removed beside it takes its mark with it.
   */
  readonly marks: Marks
  readonly refusal: Refused | null
  /** The size a deck is read up to, where that is what refused it. */
  readonly bound: number
}

const NOTHING: Told = {
  marks: { at: new Map(), fields: new Map(), whole: [] },
  refusal: null,
  bound: 0,
}

/** What one deck tab holds. */
export interface Held {
  /** The file this tab opened on, which is the identity it keeps. */
  readonly id: string
  /** The deck as the window draws it: the state it is in, and what it stands at. */
  shown(): Editing
  /** The cards, as the window holds them. */
  deck(): Deck
  /** The same, as the grid draws them, each under the stencil that cuts it. */
  drawn(): readonly Drawn[]
  /** The stencils a card may be cut by. */
  cuts(): readonly Cut[]
  /** What is wrong with the file, against the card it stands on. */
  marks(): Marks
  /** What the whole file was refused for, in words a person reads. */
  saying(): string
  /** A card cut by that stencil, named by what stands in its heading. */
  adds(
    name: string,
    stencil: string,
    values: readonly { field: string; text: string }[],
  ): void
  removes(card: string): void
  moves(card: string, at: string | null): void
  /**
   * One value of one card as it now reads. The field a card is named by lands
   * in the heading, which is the one place that field is written.
   */
  writes(card: string, field: string, text: string): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(id: string): void
}

export function decking(cards: Cards, host: Host) {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, Told>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()
  /** The stencils of the vault, as they were last listed. */
  const offers = shallowRef<readonly Offer[]>([])

  const store = editing({
    read: async (path) => {
      const answer = await cards.readDeck(path)
      const deck = answer.deck ? deckOf(answer.deck) : null
      told.set(path, {
        marks: marksOf(
          answer.deck?.problems ?? [],
          deck?.cards.map((card) => card.id) ?? [],
          [],
        ),
        refusal: answer.refusal,
        bound: answer.bound,
      })
      if (answer.deck) titles.set(path, answer.deck.title)
      if (answer.refusal !== null) return { body: '', refusal: answer.refusal }
      return { body: deck ? bodyOf(deck) : '', refusal: null, at: answer.at }
    },
    write: async (path, body, seen) => {
      const deck = deckIn(body)
      const answer = await cards.writeDeck(
        path,
        { preamble: deck.preamble, cards: cardsOf(deck), tail: deck.tail },
        seen?.at ?? null,
      )
      told.set(path, {
        marks: (told.get(path) ?? NOTHING).marks,
        refusal: answer.refusal,
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

  /** The last string a deck was read out of, and what it came to. */
  const parsed = new Map<string, { body: string; deck: Deck }>()

  /** The deck one tab is showing, read out of the string the store holds. */
  const deckAt = (id: string): Deck => {
    const body = store.shown(id).body
    const held = parsed.get(id)
    if (held && held.body === body) return held.deck
    const deck = deckIn(body)
    parsed.set(id, { body, deck })
    return deck
  }

  /** A deck as it now stands, written back into the store. */
  const turns = (id: string, deck: Deck): void => {
    const body = bodyOf(deck)
    parsed.set(id, { body, deck })
    store.typed(id, body)
  }

  /** The stencils of the vault, asked for again. */
  const lists = async (): Promise<void> => {
    try {
      offers.value = (await cards.stencils()).stencils
    } catch {
      // The stencils the window last heard of stand, and a card is cut by one
      // of them until the vault answers again.
    }
  }

  /** What one file was refused for, in words a person reads. */
  const sayingOf = (path: string): string => {
    const said = told.get(path) ?? NOTHING
    if (said.refusal === 'deckTooLarge') return words.tooLarge(said.bound)
    if (said.refusal === 'notADeck') return words.notADeck
    return said.refusal === null ? '' : words.refused
  }

  const held = (id: string): Held => ({
    id,
    shown: () => store.shown(id),
    deck: () => deckAt(id),
    drawn: () => drawnOf(deckAt(id), offers.value),
    cuts: () => cutsOf(offers.value),
    marks: () => (told.get(store.where(id)) ?? NOTHING).marks,
    saying: () => sayingOf(store.where(id)),
    adds: (name, stencil, values) =>
      turns(id, added(deckAt(id), name, stencil, pathOfCut(offers.value, stencil), values)),
    removes: (card) => turns(id, removed(deckAt(id), card)),
    moves: (card, at) => turns(id, carried(deckAt(id), card, at)),
    writes: (card, field, text) => {
      const deck = deckAt(id)
      const cut = deck.cards.find((one) => one.id === card)?.stencil ?? ''
      if (names(offers.value, cut, field)) return turns(id, named(deck, card, text))
      turns(id, filled(deck, card, field, text))
    },
    keep: () => store.keep(id),
    take: () => store.take(id),
    /** The tab stands until the deck says the write is done, and goes then. */
    shuts: (tab) => {
      void store.shut(id).then((gone) => {
        if (!gone) return
        parsed.delete(id)
        host.closes(tab)
      })
    },
  })

  /** What a deck tab is called: the title the file carries, or the file itself. */
  const called = (path: string): string => titles.get(path) || (path.split('/').pop() ?? path)

  /** The tab holding a deck lets go of it, wherever the window draws it. */
  const shuts = (id: string): void => {
    const tab = host.each<Held>(DECK).find((one) => one.held.id === id)
    tab?.held.shuts(tab.id)
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
   * A deck tab as the window keeps it. A deck is its own tab, filed under the
   * path it opened at, so the same deck asked for twice is the tab it has.
   */
  const kind: Kind<Held> = {
    kind: DECK,
    opens: (path) => {
      store.open(path)
      void lists()
      return held(path)
    },
    called: (one) => called(store.where(one.id)),
    marked: (one) => markOf(store.shown(one.id).state),
    draws: DeckTab,
    identity: (path) => path,
    shuts: (one, id) => {
      one.shuts(id)
      return false
    },
    // What an open deck owes at the quit is written by the quit, which the
    // window waits for.
    gone: () => {},
  }

  /** A deck put in front of the person, in a tab of its own. */
  const shows = (path: string, title = ''): void => {
    if (title) titles.set(path, title)
    void host.opens(DECK, path)
  }

  /** The vault changed: every open deck hears it, and the stencils are listed again. */
  const changed = (paths: readonly string[], renamed: readonly Went[] = []): void => {
    store.changed(paths, renamed)
    void lists()
  }

  return {
    kind,
    held,
    shows,
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
