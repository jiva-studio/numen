/**
 * The decks and the stencils a vault holds, as the window asks for them and as
 * they come back.
 *
 * A file is read whole or not at all, and what could not be made sense of in
 * it travels beside it as a problem against the card or the face it stands on.
 * A write presents what the read gave, so a file that moved under this window
 * is answered and not overwritten.
 */
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
   */
  readonly section: number | null
  /**
   * The line its heading says, with the mark taken off. It is not what the card
   * is called: a write throws it away and reads it again from the first field,
   * and it travels for the one card that cannot be read again — a card whose
   * stencil is missing, where nothing can say which field is first.
   */
  readonly heading: string
  /** The stencil it is cut by, as the wikilink beneath its heading names it. */
  readonly stencil: string
  /**
   * Where that stencil is filed, as the wikilink resolves in the vault. Empty
   * for a card naming none and for a name that reaches no note.
   */
  readonly stencilAt: string
  /** The prose between that wikilink and the first field. */
  readonly lead: string
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
  readonly lead: string
}

/** A deck as the vault reads it. */
export interface VaultDeck {
  readonly path: string
  readonly title: string
  /** The prose below the frontmatter and above the first section or card. */
  readonly preamble: string
  readonly cards: readonly VaultCard[]
  /** The sections, in the order they stand in the note. */
  readonly sections: readonly VaultSection[]
  /** What the file ends with once the last value has been read. */
  readonly tail: string
  readonly problems: readonly Problem[]
}

/** One way a stencil shows a card. */
export interface VaultFace {
  readonly name: string
  /** The prose between the face's heading and its first side. */
  readonly lead: string
  readonly front: string
  readonly back: string
}

/** A stencil as the vault reads it. */
export interface VaultStencil {
  readonly path: string
  readonly title: string
  readonly fields: readonly string[]
  /** The prose below the frontmatter and above the first face. */
  readonly preamble: string
  readonly faces: readonly VaultFace[]
  /** What the file ends with once the last side has been read. */
  readonly tail: string
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
