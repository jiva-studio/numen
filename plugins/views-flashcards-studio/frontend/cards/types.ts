import type { Result } from '@numen/plugin-sdk'
import type { ErrorCode } from './wire/errors'
import type { Surrounds } from './lib/surrounds'

/** Where a deck, a card model or a preset was filed, or why none was made. */
export type CreateResult = Result<{ readonly path: string }, ErrorCode>

/** One card model as the list of them names it. */
export interface CardModelSummary {
  readonly path: string
  readonly title: string
  /** The names of the fields, in the order a person is asked for them. */
  readonly fields: readonly string[]
}

/**
 * What kind of problem a card model or a deck has. The list is closed: a mark is drawn
 * by what is wrong with the card or the template it stands against.
 */
export type ProblemKind =
  | 'fieldDeclaredTwice'
  | 'cardModelWithoutFields'
  | 'templateMissingASide'
  | 'placeholderUndeclared'
  | 'cardWithoutACardModel'
  | 'cardModelIsNotOne'
  | 'cardIdCarriedTwice'
  | 'fieldWrittenTwice'
  | 'fieldNotRenamed'
  | 'unknown'

/**
 * Something in a file that could not be acted on and was not guessed at.
 */
export interface DeckProblem {
  readonly kind: ProblemKind
  /** The card it stands against, counted from the first, or nothing. */
  readonly card: number | null
  /** The template it stands against, counted from the first, or nothing. */
  readonly template: number | null
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
  readonly cardId: string
  /**
   * Where the section it is placed under stands among the deck's, counting from the first.
   */
  readonly sectionIndex: number | null
  /**
   * The line its heading says, with the card id taken off.
   */
  readonly heading: string
  /**
   * The card model it is cut by, as the wikilink beneath its heading names it.
   */
  readonly cardModelLink: string
  /**
   * Where that link lands in the vault. Empty for a card naming none.
   */
  readonly cardModelPath: string
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
  readonly problems: readonly DeckProblem[]
}

/** One way a card model shows a card. */
export interface VaultTemplate {
  readonly name: string
  /** The prose between the template's heading and its first side. */
  readonly preamble: string
  readonly front: string
  readonly back: string
}

/** A card model as the vault reads it. */
export interface VaultCardModel extends Surrounds {
  readonly path: string
  readonly title: string
  readonly fields: readonly string[]
  readonly templates: readonly VaultTemplate[]
  readonly problems: readonly DeckProblem[]
}

/**
 * Why a deck or a card model was not read or not written. Changed is a file that
 * moved past the caller, which stops a write.
 */
export type CardsFailure = ErrorCode | 'changed'

/** Why a deck was not read or not written, with the size that stopped a read. */
export interface DeckFailure {
  readonly code: CardsFailure
  /** The size a deck is read up to, in bytes. */
  readonly bound: number
}

/** What reading a deck came back with. */
export type DeckReadResult = Result<
  {
    readonly deck: VaultDeck
    /** The file it came out of, to present at the next write. */
    readonly at: string
  },
  DeckFailure
>

/** What writing a deck came back with. */
export type DeckWriteResult = Result<{ readonly at: string }, DeckFailure>

/** What reading a card model came back with. */
export type CardModelReadResult = Result<
  { readonly cardModel: VaultCardModel; readonly at: string },
  CardsFailure
>

/** What writing a card model came back with. */
export type CardModelWriteResult = Result<{ readonly at: string }, CardsFailure>

/** One deck a rename did not reach, which keeps the heading it had. */
export interface UnwrittenDeck {
  readonly path: string
  /** Why it was not reached, in the words to show. */
  readonly text: string
}

/** A field renamed, and what it reached. */
export interface RenamedField {
  /** The decks a heading was rewritten in, by path. */
  readonly decks: readonly string[]
  /** How many headings were rewritten, over all those decks. */
  readonly cards: number
  readonly notWritten: readonly UnwrittenDeck[]
  readonly at: string
}

/** What renaming a field came back with. */
export type FieldRenameResult = Result<RenamedField, CardsFailure>

/**
 * Deck persistence and lifecycle operations.
 */
export interface DeckService {
  /** A deck of no cards, filed in that folder under a name made from the title. */
  createDeck(title: string, folder: string): Promise<CreateResult>
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
    observed: string | null,
  ): Promise<DeckWriteResult>
}

/**
 * Card model metadata and schema operations.
 */
export interface CardModelService {
  /** Every card model in the vault, by what it is called and what it asks for. */
  cardModels(limit?: number): Promise<{ cardModels: readonly CardModelSummary[]; held: number }>
  /**
   * A card model declaring those fields and showing no template, the same way.
   */
  createCardModel(title: string, folder: string, fields: readonly string[]): Promise<CreateResult>
  /**
   * A field of a card model under another name, wherever that name is written.
   */
  renameField(
    path: string,
    field: { from: string; to: string },
    observed: string | null,
  ): Promise<FieldRenameResult>
  readCardModel(path: string): Promise<CardModelReadResult>
  /** Fields and templates into a card model, making the file where there is none. */
  writeCardModel(
    path: string,
    cardModel: {
      fields: readonly string[]
      preamble: string
      templates: readonly VaultTemplate[]
      tail: string
    },
    observed: string | null,
  ): Promise<CardModelWriteResult>
}

export interface Cards extends DeckService, CardModelService {}
