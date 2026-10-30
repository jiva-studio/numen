/**
 * The JSON messages the frontend exchanges with the backend of this plugin under the
 * `numen.flashcards` service.
 *
 * A request field is optional: a field left out is the zero value. A response field is present,
 * except a message-typed field or an optional scalar, which is absent when it has no value.
 * An enumeration is a string; the empty string is the unspecified value. A response that can fail
 * on a note carries the failure in `error`.
 */

/** ErrorCode names why a read or a write of a note failed. It is carried in the `error` field of a response; an empty value is no error. */
export const ErrorCode = {
  MISSING: 'missing',
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
  STALE: 'stale',
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
  /** Limit is the largest number of card models to return; a value of zero or less selects the maximum. */
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
  isUnlevelled: boolean
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
  isUnlevelled: boolean
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
  isUnlevelled: boolean
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
  /** IsUnlevelled reports that the file was written and the index did not follow. */
  isUnlevelled: boolean
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
  /** Bound is the size limit in bytes that a deck failed against; zero for any other outcome. */
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
  /** Bound is the size limit in bytes that a deck failed against; zero for any other outcome. */
  bound: number
  isUnlevelled: boolean
}

/** Goal is what a preset schedules toward. */
export const Goal = {
  UNSPECIFIED: '',
  MINUTES_A_DAY: 'minutes_a_day',
  RETENTION: 'retention',
  BY_DATE: 'by_date',
} as const
export type Goal = (typeof Goal)[keyof typeof Goal]

/** Rule is how the interval of a learned card is set. */
export const Rule = {
  UNSPECIFIED: '',
  INTERVAL: 'interval',
  RETENTION: 'retention',
} as const
export type Rule = (typeof Rule)[keyof typeof Rule]

/** BudgetUnit is what the daily budget counts. */
export const BudgetUnit = {
  UNSPECIFIED: '',
  CARDS: 'cards',
  SHOWS: 'shows',
} as const
export type BudgetUnit = (typeof BudgetUnit)[keyof typeof BudgetUnit]

/** StopReason is why a preset schedules no card for the day. */
export const StopReason = {
  UNSPECIFIED: '',
  NOTHING: 'nothing',
  NO_MINUTES: 'no_minutes',
  NO_CARDS: 'no_cards',
  NO_DAY: 'no_day',
  PAST_DAY: 'past_day',
  NO_LOAD: 'no_load',
  NO_WEEK: 'no_week',
} as const
export type StopReason = (typeof StopReason)[keyof typeof StopReason]

/** BudgetName is a limit that closes a day of review. */
export const BudgetName = {
  UNSPECIFIED: '',
  MINUTES_A_DAY: 'minutes_a_day',
  NEW_A_DAY: 'new_a_day',
  REVIEWS_A_DAY: 'reviews_a_day',
  BY_DATE: 'by_date',
  BACKLOG: 'backlog',
  PAUSED: 'paused',
} as const
export type BudgetName = (typeof BudgetName)[keyof typeof BudgetName]

/** Settings is the scheduling settings of a preset. Load maps a weekday name to the cards due that day. */
export interface Settings {
  goal: Goal
  byDate: string
  minutesADay: number
  newADay: number
  reviewsADay: number
  retention: number
  hasEvenLoad: boolean
  counts: BudgetUnit
  backlog: number
  load: Record<string, number>
  learned: Rule
  interval: number
}

/** Preset is a preset note with its settings and the reasons it schedules nothing. */
export interface Preset {
  path: string
  title: string
  settings?: Settings
  problems: string[]
  stops: StopReason
  stopsOn: StopReason
}

/** Curve is the projection of a preset over the range of its goal. */
export interface Curve {
  goal: Goal
  grid: number[]
  days: string[]
  at: Point[]
  now?: CurvePoint
  suggested?: CurvePoint
  decks: number
  cards: number
  overdue: number
  unbegun: number
}

/** Point is one sample of a projection curve. Learns is absent when the learned rule has no count. */
export interface Point {
  reviews: number
  minutes: number
  retained: number
  due: number
  through: number
  isEnough: boolean
  closed: BudgetName[]
  clears: number
  backlog: number[]
  learned: number
  learns?: number
  short: number
}

/** CurvePoint is a marked position on a projection curve. */
export interface CurvePoint {
  at: number
  value: number
  day: string
}

/** GetDeckPresetRequest is the request of the GetDeckPreset method. */
export interface GetDeckPresetRequest {
  deck?: string
}

/** GetDeckPresetResponse is the response of the GetDeckPreset method. */
export interface GetDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
  at?: FileStat
  bounds?: SettingsBounds
}

/** ListPresetsRequest is the request of the ListPresets method. */
export type ListPresetsRequest = Record<string, never>

/** ListPresetsResponse is the response of the ListPresets method. */
export interface ListPresetsResponse {
  presets: PresetSummary[]
}

/** PresetSummary is the listing entry of a preset. */
export interface PresetSummary {
  path: string
  title: string
}

/** CreatePresetRequest is the request of the CreatePreset method. */
export interface CreatePresetRequest {
  title?: string
  path?: string
}

/** CreatePresetResponse is the response of the CreatePreset method. */
export interface CreatePresetResponse {
  path: string
  error?: ErrorCode
  isUnlevelled: boolean
}

/** ScheduleDeckRequest is the request of the ScheduleDeck method. */
export interface ScheduleDeckRequest {
  deck?: string
  preset?: string
  baseline?: FileStat
}

/** ScheduleDeckResponse is the response of the ScheduleDeck method. */
export interface ScheduleDeckResponse {
  error?: ErrorCode
  at?: FileStat
  isUnlevelled: boolean
}

/** ReadPresetRequest is the request of the ReadPreset method. */
export interface ReadPresetRequest {
  path?: string
}

/** ReadPresetResponse is the response of the ReadPreset method. */
export interface ReadPresetResponse {
  preset?: Preset
  error?: ErrorCode
  at?: FileStat
  bounds?: SettingsBounds
}

/** SettingsBounds is the range each numeric setting of a preset may take. */
export interface SettingsBounds {
  minutesADay?: Bounds
  newADay?: Bounds
  reviewsADay?: Bounds
  retention?: Bounds
  backlog?: Bounds
  interval?: Bounds
  load?: Bounds
}

/** WritePresetRequest is the request of the WritePreset method. */
export interface WritePresetRequest {
  path?: string
  settings?: Settings
  baseline?: FileStat
}

/** WritePresetResponse is the response of the WritePreset method. */
export interface WritePresetResponse {
  error?: ErrorCode
  at?: FileStat
  isUnlevelled: boolean
}

/** ComputeCurveRequest is the request of the ComputeCurve method. */
export interface ComputeCurveRequest {
  path?: string
  settings?: Settings
}

/** ComputeCurveResponse is the response of the ComputeCurve method. */
export interface ComputeCurveResponse {
  curve?: Curve
}

/** Rating is how well a card was recalled. */
export const Rating = {
  UNSPECIFIED: '',
  AGAIN: 'again',
  HARD: 'hard',
  GOOD: 'good',
  EASY: 'easy',
} as const
export type Rating = (typeof Rating)[keyof typeof Rating]

/** VaultCardsDue is what one vault has due for review. Unread holds the reason the vault could not be counted. */
export interface VaultCardsDue {
  id: string
  name: string
  path: string
  reviewCards: number
  due: number
  new: number
  decks: DeckCardsDue[]
  presets: PresetCardsDue[]
  unread: string
  isIndexing: boolean
}

/** DeckCardsDue is what one deck has due for review. */
export interface DeckCardsDue {
  deck: string
  reviewCards: number
  due: number
  new: number
  reviewed: number
  learned: number
  unbegun: number
}

/** PresetCardsDue is the review count of the decks of one preset and what the current day has used. */
export interface PresetCardsDue {
  preset: string
  title: string
  decks: number
  cards: number
  pendingReviews: number
  pendingNew: number
  reviewed: number
  /** DurationMs is the time spent reviewing in the current day in milliseconds. */
  durationMs: number
  reviewedNew: number
  reviewedReviews: number
  new: number
  reviews: number
  minutes: number
  closesNew: string
  closesReviews: string
  closesMinutes: string
  closesBacklog: string
  stopsOn: StopReason
}

/** Intervals is the interval in seconds that each rating would give a card. */
export interface Intervals {
  again: number
  hard: number
  good: number
  easy: number
}

/** SessionCard is one card of a review session, laid out for one template. */
export interface SessionCard {
  deck: string
  section: string
  cardId: string
  template: string
  heading: string
  front: string
  back: string
  isNew: boolean
  due: string
  intervals?: Intervals
}

/** WatchCardsDueRequest is the request of the WatchCardsDue method. */
export type WatchCardsDueRequest = Record<string, never>

/** WatchCardsDueResponse is the response of the WatchCardsDue method. */
export interface WatchCardsDueResponse {
  /** Vaults lists every known vault; sent first and again while no count has arrived. */
  vaults: VaultCardsDue[]
  day: string
  /** Counted is the count of one vault, sent as soon as it is worked out. */
  counted?: VaultCardsDue
}

/** StartSessionRequest is the request of the StartSession method. */
export interface StartSessionRequest {
  vault?: string
  deck?: string
  /** Preset selects the decks of one preset; the path may be empty for the default preset. Absent selects no preset. */
  preset?: string
}

/** StartSessionResponse is the response of the StartSession method. */
export interface StartSessionResponse {
  run: string
  cards: SessionCard[]
  unwritten: string[]
  skipped: number
}

/** RecordReviewRequest is the request of the RecordReview method. */
export interface RecordReviewRequest {
  vault?: string
  run?: string
  cardId?: string
  template?: string
  rating?: Rating
  /** DurationMs is the time spent on the card in milliseconds. */
  durationMs?: number
}

/** RecordReviewResponse is the response of the RecordReview method. */
export interface RecordReviewResponse {
  answer: string
}

/** UndoReviewRequest is the request of the UndoReview method. */
export interface UndoReviewRequest {
  vault?: string
  run?: string
  answer?: string
}

/** UndoReviewResponse is the response of the UndoReview method. */
export type UndoReviewResponse = Record<string, never>

/** ListReviewDaysRequest is the request of the ListReviewDays method. */
export interface ListReviewDaysRequest {
  vault?: string
  from?: string
  to?: string
}

/** ListReviewDaysResponse is the response of the ListReviewDays method. */
export interface ListReviewDaysResponse {
  days: ReviewDay[]
  due: ReviewDay[]
  streak: number
  reviewed: number
}

/** ReviewDay is the review counts of one day. Tested and Recalled count the cards whose retention was measured. */
export interface ReviewDay {
  day: string
  reviewed: number
  again: number
  hard: number
  good: number
  easy: number
  tested: number
  recalled: number
}

/** GetDeckNeighbourhoodRequest is the request of the GetDeckNeighbourhood method. */
export interface GetDeckNeighbourhoodRequest {
  vault?: string
  deck?: string
}

/** GetDeckNeighbourhoodResponse is the response of the GetDeckNeighbourhood method. */
export interface GetDeckNeighbourhoodResponse {
  notes: DeckNeighbour[]
  unread: number
}

/** DeckNeighbour is a note a deck links to or is linked from. Address is the link target as the deck spells it. */
export interface DeckNeighbour {
  address: string
  path: string
  title: string
  body: string
  label: string
  isPointedAt: boolean
  isAmbiguous: boolean
  error?: ErrorCode
}

/** WatchReloadsRequest is the request of the WatchReloads method. */
export type WatchReloadsRequest = Record<string, never>

/** WatchReloadsResponse is the response of the WatchReloads method. */
export interface WatchReloadsResponse {
  /** Reload is true when the vault changed; false messages only show the stream is open. */
  reload: boolean
}

/** GetVaultDeckPresetRequest is the request of the GetVaultDeckPreset method. */
export interface GetVaultDeckPresetRequest {
  vault?: string
  deck?: string
}

/** GetVaultDeckPresetResponse is the response of the GetVaultDeckPreset method. */
export interface GetVaultDeckPresetResponse {
  preset?: Preset
  error?: ErrorCode
}
