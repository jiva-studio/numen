/**
 * The decks the window has open, and what one deck tab holds.
 *
 * A deck is saved the way a note is: one store for the whole window, the same
 * interval, and the same two answers where the file moved past what was read.
 * What travels between the store and the vault is the cards of the deck, and
 * the string the store is dirty against is those cards written out.
 */
import { computed, shallowRef } from 'vue'
import type { Cut, Drawn, PlexShowing } from '@numen/ui'
import type { Cards, Offer, Problem, Refused, Went } from '../core'
import type { Store } from '../doing'
import { editing, type Editing } from '../note/editing'
import { markOf } from '../note/tab'
import type { Host, Kind } from '../windowing'
import type { Putting } from '../putting'
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
  sameDeck,
  sameMarks,
  sameOffers,
  type Deck,
  type Marks,
} from './model'
import { WORDS as words } from './words'

/** What the vault said about one file the last time it was read or written. */
interface Told {
  /**
   * What is wrong with the file, in the order the cards were read in. Which
   * card each stands on is decided against the deck the grid is drawing, so a
   * card the window is holding through a re-read keeps its mark.
   */
  readonly problems: readonly Problem[]
  /** What the last read of the file was refused for. */
  readonly reading: Refused | null
  /** What the last write of it was refused for. */
  readonly writing: Refused | null
  /** The size a deck is read up to, where that is what refused it. */
  readonly bound: number
}

const NOTHING: Told = { problems: [], reading: null, writing: null, bound: 0 }

/** What one deck tab holds. */
export interface Held {
  /** The identity this deck opened under, which its tab keeps wherever it goes. */
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
   * in the heading, which is the one place that field is written; a card
   * writing a field twice is written where `nth` counts off under it.
   */
  writes(card: string, field: string, nth: number, text: string): void
  /** The person keeps what they have written, over whatever the file holds. */
  keep(): void
  /** The person takes what the file holds. */
  take(): void
  /** The tab is closing, and what is unwritten goes to the file first. */
  shuts(id: string): void
}

export function decking(cards: Cards, host: Host, puts: Putting) {
  /** What the vault last said about each file, under the path it is filed at. */
  const told = new Map<string, Told>()
  /** What each file is called, as the vault last read it. */
  const titles = new Map<string, string>()
  /** The stencils of the vault, as they were last listed. */
  const offers = shallowRef<readonly Offer[]>([])
  /** Whether the last listing of the stencils answered. */
  let listedOk = true

  const store = editing({
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
      return { body: deck ? bodyOf(deck) : '', refusal: null, at: answer.at }
    },
    write: async (path, body, seen) => {
      const deck = deckIn(body)
      const answer = await cards.writeDeck(
        path,
        { preamble: deck.preamble, cards: cardsOf(deck), tail: deck.tail },
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

  /** The last string a deck was read out of, and what it came to. */
  const parsed = new Map<string, { body: string; deck: Deck }>()

  /** The deck one tab is showing, read out of the string the store holds. */
  const deckAt = (id: string): Deck => {
    const body = store.shown(id).body
    const held = parsed.get(id)
    if (held && held.body === body) return held.deck
    // A file read again carries fresh identities for the same cards, so the
    // string it comes back as differs from the string that went out. A deck
    // reading as the one on screen leaves that one standing, and the card a
    // person is typing into is not drawn again.
    const read = deckIn(body)
    const deck = held && sameDeck(held.deck, read) ? held.deck : read
    parsed.set(id, { body, deck })
    return deck
  }

  /**
   * What one tab was last drawn as, against the deck and the stencils it was
   * drawn from. A deck that stands is drawn under the tiles it already has.
   */
  const grids = new Map<
    string,
    { deck: Deck; offers: readonly Offer[]; drawn: readonly Drawn[] }
  >()

  const drawnAt = (id: string): readonly Drawn[] => {
    const deck = deckAt(id)
    const held = grids.get(id)
    if (held && held.deck === deck && held.offers === offers.value) return held.drawn
    const drawn = drawnOf(deck, offers.value)
    grids.set(id, { deck, offers: offers.value, drawn })
    return drawn
  }

  /** The problems one tab was last marked from, and the marks that came of it. */
  const marked = new Map<string, { problems: readonly Problem[]; marks: Marks }>()

  /**
   * What is wrong with a file, against the card the grid is drawing. A problem
   * carries where it stood in the file it was read from, so it is put against
   * a card once, when the reading it came in on is the newest one: a card
   * carried elsewhere in the order or a card removed beside it takes its mark
   * with it from there.
   */
  const marksAt = (id: string): Marks => {
    const problems = (told.get(store.where(id)) ?? NOTHING).problems
    const held = marked.get(id)
    if (held && held.problems === problems) return held.marks
    const read = marksOf(
      problems,
      deckAt(id).cards.map((card) => card.id),
      [],
    )
    // Marks saying what the last ones said leave what is drawn against the
    // file standing.
    const marks = held && sameMarks(held.marks, read) ? held.marks : read
    marked.set(id, { problems, marks })
    return marks
  }

  /** The stencils a card may be cut by, made again where the list changed. */
  const cuts = computed(() => cutsOf(offers.value))

  /** A deck as it now stands, written back into the store. */
  const turns = (id: string, deck: Deck): void => {
    const body = bodyOf(deck)
    parsed.set(id, { body, deck })
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

  /** The words one refusal is put in, and nothing for one this file has none for. */
  const whyOf = (refusal: Refused | null, bound: number): string | null => {
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

  const held = (id: string): Held => ({
    id,
    shown: () => store.shown(id),
    deck: () => deckAt(id),
    drawn: () => drawnAt(id),
    cuts: () => cuts.value,
    marks: () => marksAt(id),
    saying: () => sayingOf(id),
    adds: (name, stencil, values) =>
      turns(id, added(deckAt(id), name, stencil, pathOfCut(offers.value, stencil), values)),
    removes: (card) => turns(id, removed(deckAt(id), card)),
    moves: (card, at) => turns(id, carried(deckAt(id), card, at)),
    writes: (card, field, nth, text) => {
      const deck = deckAt(id)
      const at = deck.cards.find((one) => one.id === card)?.stencilAt ?? ''
      if (!names(offers.value, at, field)) return turns(id, filled(deck, card, field, nth, text))
      // The field a card is named by stands first among the boxes drawn under
      // it, and that box is the heading. The ones after it are the values the
      // card writes under a heading of its own, counted off from the first.
      if (nth === 1) return turns(id, named(deck, card, text))
      turns(id, filled(deck, card, field, nth - 1, text))
    },
    keep: () => store.keep(id),
    take: () => store.take(id),
    /** The tab stands until the deck says the write is done, and goes then. */
    shuts: (tab) => {
      const path = store.where(id)
      void store.shut(id).then((gone) => {
        if (!gone) return
        parsed.delete(id)
        grids.delete(id)
        marked.delete(id)
        forgets(path)
        host.closes(tab)
      })
    },
  })

  /**
   * What the vault said about a file no tab of this window stands at any
   * longer. A second tab standing there keeps it.
   */
  const forgets = (path: string): void => {
    if (store.all().some((one) => store.where(one) === path)) return
    told.delete(path)
    titles.delete(path)
  }

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
    const standing = tabbed.value.get(path) ?? minting.get(path)
    if (standing) return standing
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
  const kind: Kind<Held> = {
    kind: DECK,
    opens: (id) => {
      const path = minted.get(id) ?? id
      store.open(id, path)
      minting.delete(path)
      minted.delete(id)
      void lists()
      return held(id)
    },
    called: (one) => called(store.where(one.id)),
    marked: (one) => markOf(store.shown(one.id).state),
    draws: DeckTab,
    identity: (id) => id,
    // A tab back on screen is a tab a person is about to draw cards in, so a
    // listing that never answered is asked for again.
    shown: () => listsAgain(),
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
    void (showing === 'beside' ? host.beside(DECK, id) : host.opens(DECK, id))
  }

  // The editor of a deck, which is the grid of its cards. A card stands on no
  // line of prose, so a deck asked for at a place inside it opens whole.
  puts.holds('deck', shows)

  /**
   * The vault changed: every open deck hears it, and the stencils are listed
   * again. The list is what a deck tab draws its cards under, so a window
   * holding no deck asks for none.
   */
  const changed = (paths: readonly string[], renamed: readonly Went[] = []): void => {
    // What the vault said about a file is filed under that file, so a file
    // that moved takes it along.
    for (const went of renamed) {
      const said = told.get(went.from)
      if (said) told.set(went.to, said)
      told.delete(went.from)
      const title = titles.get(went.from)
      if (title !== undefined) titles.set(went.to, title)
      titles.delete(went.from)
    }
    store.changed(paths, renamed)
    if (store.all().length > 0) void lists()
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
