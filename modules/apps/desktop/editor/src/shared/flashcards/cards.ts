/**
 * The decks and the stencils a vault holds, as the window asks for them and as
 * they come back.
 *
 * A file is read whole or not at all, and what could not be made sense of in
 * it travels beside it as a problem against the card or the face it stands on.
 * A write presents what the read gave, so a file that moved under this window
 * is answered and not overwritten.
 */
import { createClient } from '@connectrpc/connect'
import { CardsService, Fault as Faults } from '@numen/protocol'
import type {
  Card as CardMessage,
  Deck as DeckMessage,
  Problem as ProblemMessage,
  Stencil as StencilMessage,
} from '@numen/protocol'
import { transport } from '@numen/wire'
import { fingerprint, refusalIn, staleIn, stamp } from '../answers'
import type { MakeResult, RefusalReason } from '../core'

/** One stencil as the list of them names it. */
export interface StencilSummary {
  readonly path: string
  readonly title: string
  /** The names of the fields, in the order a person is asked for them. */
  readonly fields: readonly string[]
}

/**
 * What is wrong with a stencil or a deck. The list is closed: a mark is drawn
 * by what is wrong with the card or the face it stands against.
 */
export type Fault =
  | 'fieldDeclaredTwice'
  | 'stencilWithoutFields'
  | 'faceMissingASide'
  | 'placeholderUndeclared'
  | 'cardWithoutAStencil'
  | 'stencilIsNotOne'
  | 'markCarriedTwice'
  | 'fieldWrittenTwice'
  | 'fieldNotRenamed'
  | 'unknown'

/**
 * Something in a file that could not be acted on and was not guessed at. The
 * file is read either way, and where it stands is what the mark is drawn on.
 */
export interface Problem {
  readonly fault: Fault
  /** The card it stands against, counted from the first, or nothing. */
  readonly card: number | null
  /** The face it stands against, counted from the first, or nothing. */
  readonly face: number | null
  /** The field's name, as the file spells it. Empty for a problem against no field. */
  readonly field: string
  /** What is wrong, in the words to show. */
  readonly text: string
}

/** What a person wrote under one of a card's fields. */
export interface Value {
  readonly field: string
  readonly text: string
}

/** One card as the vault reads it. */
export interface VaultCard {
  /**
   * What the card is, for as long as it exists, without the caret its heading
   * writes it behind. Empty for a card the application has not written yet.
   */
  readonly mark: string
  /**
   * Where the section it stands under stands among the deck's, counting from
   * the first. Nothing for a card standing before the first section.
   *
   * The window addresses a section by the identity it minted for it instead:
   * a card dragged elsewhere or a section removed renumbers every card after
   * it, and a card renumbered under a person's hands is a card drawn again.
   */
  readonly sectionIndex: number | null
  /**
   * The line its heading says, with the mark taken off. It is not what the card
   * is called: a write throws it away and reads it again from the first field,
   * and it travels for the one card that cannot be read again — a card whose
   * stencil is missing, where nothing can say which field is first.
   */
  readonly heading: string
  /**
   * The stencil it is cut by, as the wikilink beneath its heading names it: a
   * name where one picks the stencil, and `note://<identifier>` where none
   * does.
   */
  readonly stencilLink: string
  /**
   * Where that link lands in the vault. Empty for a card naming none and for a
   * link that reaches no note.
   */
  readonly stencilPath: string
  /** The prose between that wikilink and the first field. */
  readonly preamble: string
  readonly values: readonly Value[]
}

/**
 * One section of a deck as the vault reads it. It is a name and nothing else:
 * no fields, no stencil, no schedule, no mark.
 */
export interface VaultSection {
  /** What it is called, as its heading spells it. Two sections may carry one name. */
  readonly name: string
  /** The prose between its heading and its first card. */
  readonly preamble: string
}

/**
 * The prose standing around what this window edits, kept as the person left
 * it. A note is theirs, and what they wrote above the first card and below the
 * last comes back written as it went out.
 */
export interface Surrounds {
  /** The prose below the frontmatter and above the first of them. */
  readonly preamble: string
  /** What the file ends with once the last of them has been read. */
  readonly tail: string
}

/** A deck as the vault reads it. */
export interface VaultDeck extends Surrounds {
  readonly path: string
  readonly title: string
  readonly cards: readonly VaultCard[]
  /** The sections, in the order they stand in the note. */
  readonly sections: readonly VaultSection[]
  readonly problems: readonly Problem[]
}

/** One way a stencil shows a card. */
export interface VaultFace {
  readonly name: string
  /** The prose between the face's heading and its first side. */
  readonly preamble: string
  readonly front: string
  readonly back: string
}

/** A stencil as the vault reads it. */
export interface VaultStencil extends Surrounds {
  readonly path: string
  readonly title: string
  readonly fields: readonly string[]
  readonly faces: readonly VaultFace[]
  readonly problems: readonly Problem[]
}

/** What reading a deck came back with. */
export interface DeckReadResult {
  /** Null when the deck was refused. */
  readonly deck: VaultDeck | null
  readonly refusal: RefusalReason | null
  /** The file it came out of, to present at the next write. */
  readonly at: string
  /** The size a deck is read up to, in bytes. */
  readonly bound: number
}

/** What writing a deck came back with. */
export interface DeckWriteResult {
  readonly refusal: RefusalReason | null
  /** The file is no longer the one this caller read, and nothing was written. */
  readonly changed: boolean
  readonly at: string
  readonly bound: number
}

/** What reading a stencil came back with. */
export interface StencilReadResult {
  /** Null when the stencil was refused. */
  readonly stencil: VaultStencil | null
  readonly refusal: RefusalReason | null
  readonly at: string
}

/** What writing a stencil came back with. */
export interface StencilWriteResult {
  readonly refusal: RefusalReason | null
  readonly changed: boolean
  readonly at: string
}

/** One deck a rename did not reach, which keeps the heading it had. */
export interface UnwrittenDeck {
  readonly path: string
  /** Why it was not reached, in the words to show. */
  readonly text: string
}

/** What renaming a field came back with. */
export interface FieldRenameResult {
  /** The decks a heading was rewritten in, by path. */
  readonly decks: readonly string[]
  /** How many headings were rewritten, over all those decks. */
  readonly cards: number
  readonly notWritten: readonly UnwrittenDeck[]
  /** Set where nothing was renamed at all. */
  readonly refusal: RefusalReason | null
  /** The stencil is no longer the one this caller read, and nothing was renamed. */
  readonly changed: boolean
  readonly at: string
}

/**
 * The stencils and the decks of the vault this window is showing.
 *
 * Nothing here is laid out: a face travels as the markdown it was written as,
 * and putting a card's values into it is the window's.
 */
export interface Cards {
  /** Every stencil in the vault, by what it is called and what it asks for. */
  stencils(limit?: number): Promise<{ stencils: readonly StencilSummary[]; held: number }>
  /** A deck of no cards, filed in that folder under a name made from the title. */
  makeDeck(title: string, folder: string): Promise<MakeResult>
  /**
   * A stencil declaring those fields and showing no face, the same way. The
   * first field names the cards it cuts, so a stencil is made carrying one.
   */
  makeStencil(title: string, folder: string, fields: readonly string[]): Promise<MakeResult>
  /**
   * A field of a stencil under another name, wherever that name is written: in
   * the stencil's fields, in the placeholders of its faces, and as a heading in
   * every card that stencil cuts. Seen is what a read gave this caller, and a
   * stencil that moved past it comes back changed with nothing renamed.
   */
  renameField(
    path: string,
    from: string,
    to: string,
    seen: string | null,
  ): Promise<FieldRenameResult>
  readDeck(path: string): Promise<DeckReadResult>
  /**
   * Sections and cards into a deck, in the order they are given, making the
   * file where there is none. Seen is what a read gave this caller, and a file
   * that moved past it comes back changed with nothing written.
   */
  writeDeck(
    path: string,
    deck: {
      preamble: string
      cards: readonly VaultCard[]
      sections: readonly VaultSection[]
      tail: string
    },
    seen: string | null,
  ): Promise<DeckWriteResult>
  readStencil(path: string): Promise<StencilReadResult>
  /** Fields and faces into a stencil, making the file where there is none. */
  writeStencil(
    path: string,
    fields: readonly string[],
    stencil: { preamble: string; faces: readonly VaultFace[]; tail: string },
    seen: string | null,
  ): Promise<StencilWriteResult>
}

const cardsService = createClient(CardsService, transport)

/** The stencils and the decks of that vault, in the shape the window asks about them. */
export const cards: Cards = {
  stencils: async (limit) => {
    const answer = await cardsService.listStencils({ limit: limit ?? 0 })
    return { stencils: answer.stencils.map(offered), held: answer.total }
  },
  makeDeck: async (title, folder) => {
    const answer = await cardsService.createDeck({ title, path: folder })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  makeStencil: async (title, folder, fields) => {
    const answer = await cardsService.createStencil({ title, path: folder, fields: [...fields] })
    return { path: answer.path, refusal: refusalIn(answer) }
  },
  renameField: async (path, from, to, seen) => {
    const answer = await cardsService.renameStencilField({
      path,
      from,
      to,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      decks: answer.decks,
      cards: answer.cards,
      notWritten: answer.notWritten.map((one) => ({
        path: one.path,
        text: one.problem?.text ?? '',
      })),
      refusal: refusalIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  readDeck: async (path) => {
    const answer = await cardsService.readDeck({ path })
    return {
      deck: answer.deck ? decked(answer.deck) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  writeDeck: async (path, deck, seen) => {
    const answer = await cardsService.writeDeck({
      path,
      preamble: deck.preamble,
      cards: deck.cards.map(carding),
      sections: deck.sections.map((section) => ({ name: section.name, preamble: section.preamble })),
      tail: deck.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      refusal: refusalIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
      bound: Number(answer.bound),
    }
  },
  readStencil: async (path) => {
    const answer = await cardsService.readStencil({ path })
    return {
      stencil: answer.stencil ? stencilled(answer.stencil) : null,
      refusal: refusalIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
  writeStencil: async (path, fields, stencil, seen) => {
    const answer = await cardsService.writeStencil({
      path,
      fields: [...fields],
      preamble: stencil.preamble,
      faces: stencil.faces.map((face) => ({
        name: face.name,
        preamble: face.preamble,
        front: face.front,
        back: face.back,
      })),
      tail: stencil.tail,
      ...(seen === null ? {} : { seen: fingerprint(seen) }),
    })
    return {
      refusal: refusalIn(answer),
      changed: staleIn(answer),
      at: stamp(answer.at) ?? '',
    }
  },
}

/** One stencil of the list, kept as the plain value the window carries it as. */
const offered = (one: {
  path: string
  title: string
  fields: string[]
}): StencilSummary => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
})

/** A deck as the window carries it. */
const decked = (one: DeckMessage): VaultDeck => ({
  path: one.path,
  title: one.title,
  preamble: one.preamble,
  cards: one.cards.map(carded),
  sections: one.sections.map((section) => ({ name: section.name, preamble: section.preamble })),
  tail: one.tail,
  problems: one.problems.map(problem),
})

/** A stencil as the window carries it. */
const stencilled = (one: StencilMessage): VaultStencil => ({
  path: one.path,
  title: one.title,
  fields: one.fields,
  preamble: one.preamble,
  faces: one.faces.map(
    (face): VaultFace => ({
      name: face.name,
      preamble: face.preamble,
      front: face.front,
      back: face.back,
    }),
  ),
  tail: one.tail,
  problems: one.problems.map(problem),
})

const carded = (one: CardMessage): VaultCard => ({
  mark: one.mark,
  sectionIndex: one.sectionIndex ?? null,
  heading: one.heading,
  stencilLink: one.stencilLink,
  stencilPath: one.stencilPath,
  preamble: one.preamble,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/**
 * One card in the shape the schema carries it. The heading goes back as it
 * came: a write reads it again from the first field, except for the one card
 * whose stencil cannot be read, whose heading is left exactly as it stands.
 */
const carding = (one: VaultCard) => ({
  mark: one.mark,
  ...(one.sectionIndex === null ? {} : { sectionIndex: one.sectionIndex }),
  heading: one.heading,
  stencilLink: one.stencilLink,
  preamble: one.preamble,
  values: one.values.map((value) => ({ field: value.field, text: value.text })),
})

/** One problem, with where it stands kept as a number or as nothing. */
const problem = (one: ProblemMessage): Problem => ({
  fault: faulted[one.fault],
  card: one.card ?? null,
  face: one.face ?? null,
  field: one.field,
  text: one.text,
})

/** What a problem is, in the words the window uses. */
const faulted: Record<Faults, Fault> = {
  [Faults.UNSPECIFIED]: 'unknown',
  [Faults.FIELD_DECLARED_TWICE]: 'fieldDeclaredTwice',
  [Faults.STENCIL_WITHOUT_FIELDS]: 'stencilWithoutFields',
  [Faults.FACE_MISSING_A_SIDE]: 'faceMissingASide',
  [Faults.PLACEHOLDER_UNDECLARED]: 'placeholderUndeclared',
  [Faults.CARD_WITHOUT_A_STENCIL]: 'cardWithoutAStencil',
  [Faults.STENCIL_IS_NOT_ONE]: 'stencilIsNotOne',
  [Faults.MARK_CARRIED_TWICE]: 'markCarriedTwice',
  [Faults.FIELD_WRITTEN_TWICE]: 'fieldWrittenTwice',
  [Faults.FIELD_NOT_RENAMED]: 'fieldNotRenamed',
}
