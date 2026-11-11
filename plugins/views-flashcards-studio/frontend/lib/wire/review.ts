/** The JSON messages of the review of this plugin: what is due, review sessions, review days and the notes around a deck. */
import type { StopReason } from '../stop'
import type { ErrorCode } from './notes'

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
interface VaultCardsDue {
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
interface DeckCardsDue {
  deck: string
  reviewCards: number
  due: number
  new: number
  reviewed: number
  learned: number
  unbegun: number
}

/** PresetCardsDue is the review count of the decks of one preset and what the current day has used. */
interface PresetCardsDue {
  preset: string
  title: string
  decks: number
  cards: number
  pendingReviews: number
  pendingNew: number
  reviewed: number
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
interface Intervals {
  again: number
  hard: number
  good: number
  easy: number
}

/** SessionCard is one card of a review session, laid out for one template. */
interface SessionCard {
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
  vaults: VaultCardsDue[]
  day: string
  counted?: VaultCardsDue
}

/** StartSessionRequest is the request of the StartSession method. */
export interface StartSessionRequest {
  vault?: string
  deck?: string
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
interface ReviewDay {
  day: string
  reviewed: number
  again: number
  hard: number
  good: number
  easy: number
  tested: number
  recalled: number
}

/** GetDeckNeighborhoodRequest is the request of the GetDeckNeighborhood method. */
export interface GetDeckNeighborhoodRequest {
  vault?: string
  deck?: string
}

/** GetDeckNeighborhoodResponse is the response of the GetDeckNeighborhood method. */
export interface GetDeckNeighborhoodResponse {
  notes: DeckNeighbor[]
  unread: number
}

/** DeckNeighbor is a note a deck links to or is linked from. Address is the link target as the deck spells it. */
interface DeckNeighbor {
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
  reload: boolean
}
