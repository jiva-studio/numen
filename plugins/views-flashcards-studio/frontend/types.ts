import type { Result } from '@numen/plugin-sdk'

/** Where a deck, a card model or a preset was filed, or why none was made. */
type CreateResult = Result<{ readonly path: string }, ErrorCode>

/** One card model as the list of them names it. */
export interface CardModelSummary {
  readonly path: string
  readonly title: string
  readonly fields: readonly string[]
}

/**
 * The category of problem a card model or a deck has. The list is closed: a mark is
 * drawn according to the problem in the card or the template it refers to.
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
  readonly card: number | null
  readonly template: number | null
  readonly field: string
  readonly text: string
}

/** The text written under one of a card's fields. */
export interface Value {
  readonly field: string
  readonly text: string
}

/** One card as the vault reads it. */
export interface VaultCard {
  readonly cardId: string
  readonly sectionIndex: number | null
  readonly heading: string
  readonly cardModelLink: string
  readonly cardModelPath: string
  readonly preamble: string
  readonly values: readonly Value[]
}

/**
 * One section of a deck as the vault reads it.
 */
export interface VaultSection {
  readonly name: string
  readonly preamble: string
}

/** A deck as the vault reads it. */
export interface VaultDeck extends Surrounds {
  readonly path: string
  readonly title: string
  readonly cards: readonly VaultCard[]
  readonly sections: readonly VaultSection[]
  readonly problems: readonly DeckProblem[]
}

/** One way a card model shows a card. */
export interface VaultTemplate {
  readonly name: string
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
interface DeckFailure {
  readonly code: CardsFailure
  readonly bound: number
}

/** What reading a deck returned. */
type DeckReadResult = Result<
  {
    readonly deck: VaultDeck
    readonly at: string
  },
  DeckFailure
>

/** What writing a deck returned. */
type DeckWriteResult = Result<{ readonly at: string }, DeckFailure>

/** What reading a card model returned. */
type CardModelReadResult = Result<
  { readonly cardModel: VaultCardModel; readonly at: string },
  CardsFailure
>

/** What writing a card model returned. */
type CardModelWriteResult = Result<{ readonly at: string }, CardsFailure>

/** One deck a rename did not reach, which keeps the heading it had. */
interface UnwrittenDeck {
  readonly path: string
  readonly text: string
}

/** A field renamed, and what it reached. */
export interface RenamedField {
  readonly decks: readonly string[]
  readonly cards: number
  readonly notWritten: readonly UnwrittenDeck[]
  readonly at: string
}

/** What renaming a field returned. */
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
  /** Every card model in the vault, with its name and its fields. */
  cardModels(limit?: number): Promise<{ cardModels: readonly CardModelSummary[]; total: number }>
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

/** The error code a call of this plugin answers with. */
export type ErrorCode =
  | 'notFound'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyUnwritable'
  | 'unreadable'
  | 'alreadyExists'
  | 'invalidName'
  | 'unreachable'
  | 'notACardModel'
  | 'notADeck'
  | 'deckTooLarge'
  | 'notAPreset'

/**
 * The prose around the part a card tab edits, preserved unchanged. A note
 * belongs to its user, and the text above the first card and below the last is
 * written back as it was read.
 */
export interface Surrounds {
  readonly preamble: string
  readonly tail: string
}
