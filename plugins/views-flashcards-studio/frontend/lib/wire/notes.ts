/** The JSON messages of the notes of this plugin: card models and decks, and the errors both report. */

/** ErrorCode names why a read or a write of a note failed. It is carried in the `error` field of a response; an empty value is no error. */
export const ErrorCode = {
  NOT_FOUND: 'not_found',
  NOT_A_NOTE: 'not_a_note',
  NOT_TEXT: 'not_text',
  TOO_LARGE: 'too_large',
  BODY_UNWRITABLE: 'body_unwritable',
  UNREADABLE: 'unreadable',
  ALREADY_EXISTS: 'already_exists',
  INVALID_NAME: 'invalid_name',
  NOT_A_CARD_MODEL: 'not_a_card_model',
  NOT_A_DECK: 'not_a_deck',
  DECK_TOO_LARGE: 'deck_too_large',
  NOT_A_PRESET: 'not_a_preset',
  CONFLICT: 'conflict',
} as const
export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode]

/**
 * FileStat identifies a version of a file: its path, its size in bytes and its modification time in
 * nanoseconds since the epoch. Mtime is a decimal string because a nanosecond count is larger than
 * the integers a JSON number carries without loss.
 */
export interface FileStat {
  path: string
  size: number
  mtime: string
}

/** Bounds is the closed range a number may take. */
export interface Bounds {
  min: number
  max: number
}

/** ProblemKind names a defect found in a card model or a deck. */
export const ProblemKind = {
  UNSPECIFIED: '',
  FIELD_DECLARED_TWICE: 'field_declared_twice',
  CARD_MODEL_WITHOUT_FIELDS: 'card_model_without_fields',
  TEMPLATE_MISSING_A_SIDE: 'template_missing_a_side',
  PLACEHOLDER_UNDECLARED: 'placeholder_undeclared',
  CARD_WITHOUT_A_CARD_MODEL: 'card_without_a_card_model',
  CARD_MODEL_IS_NOT_ONE: 'card_model_is_not_one',
  FIELD_WRITTEN_TWICE: 'field_written_twice',
  FIELD_NOT_RENAMED: 'field_not_renamed',
  CARD_ID_CARRIED_TWICE: 'card_id_carried_twice',
} as const
export type ProblemKind = (typeof ProblemKind)[keyof typeof ProblemKind]

/** CardModel is a card model note: the fields its cards declare and the templates that lay them out. */
export interface CardModel {
  path: string
  title: string
  fields: string[]
  templates: CardTemplate[]
  problems: Problem[]
  preamble: string
  tail: string
}

/** CardModelSummary is the listing entry of a card model. */
interface CardModelSummary {
  path: string
  title: string
  fields: string[]
}

/** CardTemplate is one template of a card model; front and back are Markdown with `{{Field}}` placeholders. */
interface CardTemplate {
  name: string
  front: string
  back: string
  preamble: string
}

/** Deck is a deck note: its sections and its cards in file order. */
export interface Deck {
  path: string
  title: string
  preamble: string
  cards: Card[]
  sections: Section[]
  problems: Problem[]
  tail: string
}

/** Section is a heading of a deck that groups cards. */
interface Section {
  name: string
  preamble: string
}

/** Problem is a defect found in a card model or a deck. Card and Template are zero-based positions and are absent when the defect concerns neither. */
export interface Problem {
  kind: ProblemKind
  card?: number
  template?: number
  field: string
  text: string
}

/** Card is one card of a deck. SectionIndex is the zero-based index into the sections of the deck and is absent for a card outside any section. */
export interface Card {
  cardId: string
  sectionIndex?: number
  heading: string
  cardModelLink: string
  preamble: string
  values: Value[]
  cardModelPath?: string
}

/** Value is the text a card holds for one field of its card model. */
interface Value {
  field: string
  text: string
}

export interface ListCardModelsRequest {
  limit?: number
}

export interface ListCardModelsResponse {
  cardModels: CardModelSummary[]
  total: number
}

export interface CreateCardModelRequest {
  title?: string
  path?: string
  fields?: string[]
}

export interface CreateCardModelResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

export interface ReadCardModelRequest {
  path?: string
}

export interface ReadCardModelResponse {
  cardModel?: CardModel
  error?: ErrorCode
  at?: FileStat
}

export interface WriteCardModelRequest {
  path?: string
  fields?: string[]
  templates?: CardTemplate[]
  baseline?: FileStat
  preamble?: string
  tail?: string
}

export interface WriteCardModelResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

export interface RenameCardModelFieldRequest {
  path?: string
  from?: string
  to?: string
  baseline?: FileStat
}

export interface RenameCardModelFieldResponse {
  decks: string[]
  cards: number
  notWritten: UnwrittenDeck[]
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

/** UnwrittenDeck is a deck that a field rename did not reach, with the problem that stopped it. */
interface UnwrittenDeck {
  path: string
  problem?: Problem
}

export interface CreateDeckRequest {
  title?: string
  path?: string
}

export interface CreateDeckResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

export interface ReadDeckRequest {
  path?: string
}

export interface ReadDeckResponse {
  deck?: Deck
  error?: ErrorCode
  at?: FileStat
  bound: number
}

export interface WriteDeckRequest {
  path?: string
  preamble?: string
  cards?: Card[]
  sections?: Section[]
  baseline?: FileStat
  tail?: string
}

export interface WriteDeckResponse {
  error?: ErrorCode
  at?: FileStat
  bound: number
  isIndexOutdated: boolean
}
