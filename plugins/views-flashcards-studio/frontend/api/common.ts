/**
 * The messages shared by more than one method of the backend: the failure of a note, the stat of a file, and a range.
 *
 * A request field is optional: a field left out is the zero value. A response field is present,
 * except a message-typed field or an optional scalar, which is absent when it has no value.
 * An enumeration is a string; the empty string is the unspecified value.
 */

/** ErrorCode names why a read or a write of a note failed. It is carried in the `error` field of a response; an empty value is no error. */
export const ErrorCode = {
  NOT_FOUND: 'notFound',
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
