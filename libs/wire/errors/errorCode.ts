/** The error code a vault operation answers with. */
export type ErrorCode =
  | 'missing'
  | 'notANote'
  | 'notText'
  | 'tooLarge'
  | 'bodyUnwritable'
  | 'unreadable'
  | 'alreadyExists'
  | 'invalidName'
  | 'unreachable'
