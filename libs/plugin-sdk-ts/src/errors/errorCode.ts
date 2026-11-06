/** The error code a vault operation answers with. */
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
