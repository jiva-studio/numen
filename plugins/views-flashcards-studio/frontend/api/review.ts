/**
 * The messages of the methods that count what is due, run a review session and read the notes beside a deck.
 *
 * A request field is optional: a field left out is the zero value. A response field is present,
 * except a message-typed field or an optional scalar, which is absent when it has no value.
 * An enumeration is a string; the empty string is the unspecified value.
 */
import type { ErrorCode } from './common'
import type { Preset } from './presets'
import type { StopReason } from '../lib/stop'

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
