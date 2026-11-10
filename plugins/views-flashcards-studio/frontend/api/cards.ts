/**
 * The messages of the methods that create, read and write card models and decks.
 *
 * A request field is optional: a field left out is the zero value. A response field is present,
 * except a message-typed field or an optional scalar, which is absent when it has no value.
 * An enumeration is a string; the empty string is the unspecified value.
 */
import type { ErrorCode, FileStat, ProblemKind } from './common'

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
export interface CardModelSummary {
  path: string
  title: string
  fields: string[]
}

/** CardTemplate is one template of a card model; front and back are Markdown with `{{Field}}` placeholders. */
export interface CardTemplate {
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
export interface Section {
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
export interface Value {
  field: string
  text: string
}

/** ListCardModelsRequest is the request of the ListCardModels method. */
export interface ListCardModelsRequest {
  limit?: number
}

/** ListCardModelsResponse is the response of the ListCardModels method. */
export interface ListCardModelsResponse {
  cardModels: CardModelSummary[]
  total: number
}

/** CreateCardModelRequest is the request of the CreateCardModel method. */
export interface CreateCardModelRequest {
  title?: string
  path?: string
  fields?: string[]
}

/** CreateCardModelResponse is the response of the CreateCardModel method. */
export interface CreateCardModelResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

/** ReadCardModelRequest is the request of the ReadCardModel method. */
export interface ReadCardModelRequest {
  path?: string
}

/** ReadCardModelResponse is the response of the ReadCardModel method. */
export interface ReadCardModelResponse {
  cardModel?: CardModel
  error?: ErrorCode
  at?: FileStat
}

/** WriteCardModelRequest is the request of the WriteCardModel method. */
export interface WriteCardModelRequest {
  path?: string
  fields?: string[]
  templates?: CardTemplate[]
  baseline?: FileStat
  preamble?: string
  tail?: string
}

/** WriteCardModelResponse is the response of the WriteCardModel method. */
export interface WriteCardModelResponse {
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

/** RenameCardModelFieldRequest is the request of the RenameCardModelField method. */
export interface RenameCardModelFieldRequest {
  path?: string
  from?: string
  to?: string
  baseline?: FileStat
}

/** RenameCardModelFieldResponse is the response of the RenameCardModelField method. */
export interface RenameCardModelFieldResponse {
  decks: string[]
  cards: number
  notWritten: UnwrittenDeck[]
  error?: ErrorCode
  at?: FileStat
  isIndexOutdated: boolean
}

/** UnwrittenDeck is a deck that a field rename did not reach, with the problem that stopped it. */
export interface UnwrittenDeck {
  path: string
  problem?: Problem
}

/** CreateDeckRequest is the request of the CreateDeck method. */
export interface CreateDeckRequest {
  title?: string
  path?: string
}

/** CreateDeckResponse is the response of the CreateDeck method. */
export interface CreateDeckResponse {
  path: string
  error?: ErrorCode
  isIndexOutdated: boolean
}

/** ReadDeckRequest is the request of the ReadDeck method. */
export interface ReadDeckRequest {
  path?: string
}

/** ReadDeckResponse is the response of the ReadDeck method. */
export interface ReadDeckResponse {
  deck?: Deck
  error?: ErrorCode
  at?: FileStat
  bound: number
}

/** WriteDeckRequest is the request of the WriteDeck method. */
export interface WriteDeckRequest {
  path?: string
  preamble?: string
  cards?: Card[]
  sections?: Section[]
  baseline?: FileStat
  tail?: string
}

/** WriteDeckResponse is the response of the WriteDeck method. */
export interface WriteDeckResponse {
  error?: ErrorCode
  at?: FileStat
  bound: number
  isIndexOutdated: boolean
}
