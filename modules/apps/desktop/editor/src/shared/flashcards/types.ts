import type { MakeResult, RefusalReason } from '../note'
import type { Surrounds } from './surrounds'

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
 * Something in a file that could not be acted on and was not guessed at.
 */
export interface Problem {
  readonly fault: Fault
  /** The card it stands against, counted from the first, or nothing. */
  readonly card: number | null
  /** The face it stands against, counted from the first, or nothing. */
  readonly face: number | null
  /** The field name, as the file spells it. Empty for a problem against no field. */
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
   * What the card is, without the caret its heading writes it behind.
   */
  readonly mark: string
  /**
   * Where the section it stands under stands among the deck's, counting from the first.
   */
  readonly sectionIndex: number | null
  /**
   * The line its heading says, with the mark taken off.
   */
  readonly heading: string
  /**
   * The stencil it is cut by, as the wikilink beneath its heading names it.
   */
  readonly stencilLink: string
  /**
   * Where that link lands in the vault. Empty for a card naming none.
   */
  readonly stencilPath: string
  /** The prose between that wikilink and the first field. */
  readonly preamble: string
  readonly values: readonly Value[]
}

/**
 * One section of a deck as the vault reads it.
 */
export interface VaultSection {
  /** What it is called, as its heading spells it. */
  readonly name: string
  /** The prose between its heading and its first card. */
  readonly preamble: string
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
 */
export interface Cards {
  /** Every stencil in the vault, by what it is called and what it asks for. */
  stencils(limit?: number): Promise<{ stencils: readonly StencilSummary[]; held: number }>
  /** A deck of no cards, filed in that folder under a name made from the title. */
  makeDeck(title: string, folder: string): Promise<MakeResult>
  /**
   * A stencil declaring those fields and showing no face, the same way.
   */
  makeStencil(title: string, folder: string, fields: readonly string[]): Promise<MakeResult>
  /**
   * A field of a stencil under another name, wherever that name is written.
   */
  renameField(
    path: string,
    from: string,
    to: string,
    seen: string | null,
  ): Promise<FieldRenameResult>
  readDeck(path: string): Promise<DeckReadResult>
  /**
   * Sections and cards into a deck, in the order they are given.
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
