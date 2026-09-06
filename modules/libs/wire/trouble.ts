/**
 * What went wrong, in words a person reads. Both are clauses, read after
 * something that says which thing went wrong — a note's title, the row a
 * refusal is set beside — and a window that wants a sentence of one makes it
 * one.
 */
import { Code, ConnectError } from '@connectrpc/connect'
import { Refusal } from '@numen/protocol'

/** What is said when nothing more precise can honestly be said. */
const UNEXPECTED = 'something inside numen went wrong'

/**
 * What a call that could not be answered says. Unknown is what a connection
 * that has gone arrives as, a code nothing here produces says what is true and
 * no more, and a call the window itself stopped says nothing.
 */
const TROUBLE: Record<Code, string> = {
  [Code.Canceled]: '',
  [Code.Unknown]:
    'numen did not answer, so nothing was done — it may have stopped, and the window keeps trying',
  [Code.InvalidArgument]: 'numen could not make sense of what was asked, so nothing was done',
  [Code.DeadlineExceeded]: 'numen took too long to answer, so nothing was done',
  [Code.NotFound]: 'what was asked for is not there',
  [Code.AlreadyExists]: UNEXPECTED,
  [Code.PermissionDenied]: UNEXPECTED,
  [Code.ResourceExhausted]: UNEXPECTED,
  [Code.FailedPrecondition]: 'numen cannot do that as things now stand',
  [Code.Aborted]: UNEXPECTED,
  [Code.OutOfRange]: UNEXPECTED,
  [Code.Unimplemented]: 'this installation of numen cannot do that',
  [Code.Internal]: UNEXPECTED,
  [Code.Unavailable]: 'numen is not answering just now — try again in a moment',
  [Code.DataLoss]: UNEXPECTED,
  [Code.Unauthenticated]: UNEXPECTED,
}

/**
 * The codes the application raises only with a sentence of its own.
 *
 * Everything under them is hand-written and reads as a clause: a vault that is
 * closing, a recording a run is holding, a setting that does not read as JSON
 * and the byte it stops at. Not found is out, because half of what it carries
 * is `fs.ErrNotExist` with a path and a stat call in front of it.
 */
const CARRIES = new Set<Code>([
  Code.InvalidArgument,
  Code.FailedPrecondition,
  Code.Unavailable,
  Code.Unimplemented,
])

/**
 * What a caught fault says to the person who was waiting for the answer.
 *
 * Anything at all can be thrown, so anything at all is taken. What the window
 * threw at itself is not the person's to read either, and lands on the same
 * words as a code nothing here produces.
 */
export const troubleWords = (thrown: unknown): string => {
  const fault = ConnectError.from(thrown)
  const carried = CARRIES.has(fault.code) ? fault.rawMessage.trim() : ''
  return carried || TROUBLE[fault.code]
}

/**
 * What each refusal the schema carries says.
 *
 * Keyed by the schema itself, so a refusal added to the protocol has no words
 * until someone writes them, and nothing that reads this compiles until
 * someone does.
 */
const REFUSED: Record<Refusal, string> = {
  [Refusal.UNSPECIFIED]: 'that note could not be read, and numen did not say why',
  [Refusal.MISSING]: 'that note is not in the vault',
  [Refusal.NOT_A_NOTE]: 'that file is not a note',
  [Refusal.NOT_TEXT]: 'that file is not text',
  [Refusal.TOO_LARGE]: 'that note is longer than this reads',
  [Refusal.BODY_REFUSED]: 'a note begins below its frontmatter, and that text begins with one',
  [Refusal.UNREADABLE]: 'the frontmatter of that note cannot be read',
  [Refusal.OCCUPIED]: 'something of that name is filed there already, so nothing was written',
  [Refusal.UNNAMEABLE]: 'a note cannot be called that',
  [Refusal.NOT_A_STENCIL]: 'that note is not a stencil',
  [Refusal.NOT_A_DECK]: 'that note is not a deck',
  [Refusal.DECK_TOO_LARGE]: 'that deck is longer than this reads',
  [Refusal.NOT_A_PRESET]: 'that note is not a preset',
  [Refusal.STALE]: 'that note changed on disk, so nothing was written',
}

/** What one refusal says, and nothing where the answer was not refused. */
export const refusalWords = (refusal: Refusal | undefined): string =>
  refusal === undefined ? '' : REFUSED[refusal]
