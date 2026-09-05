/**
 * What went wrong, in words a person reads.
 *
 * Every window catches what a call threw and is answered refusals, and neither
 * is anything to put in front of a person as it stands: a class name, a stack
 * and a message written for whoever wrote the server all read the same to
 * whoever is looking at the screen. What is said here is what happened and,
 * where there is one, what they can do about it.
 *
 * The two are shaped for the two places they are drawn. A fault stands alone in
 * a block of its own, so it is sentences; a refusal is set inline beside the
 * thing refused, so it is the clause the rest of that line reads on.
 */
import { Code, ConnectError } from '@connectrpc/connect'
import { Refusal } from '@numen/protocol'

/** What is said when nothing more precise can honestly be said. */
const UNEXPECTED = 'Something inside numen went wrong.'

/**
 * What a call that could not be answered says.
 *
 * Six of these the application sends itself — invalid argument, internal, not
 * found, unavailable, failed precondition and unimplemented — and the client
 * raises cancelled, unknown and deadline exceeded on its own, unknown being
 * what a connection that has gone arrives as. The rest nothing here produces,
 * and a sentence invented for one would be a sentence nobody could act on, so
 * they say what is true and no more.
 *
 * A call the window itself stopped says nothing, because nobody but the window
 * asked for it to stop.
 */
const TROUBLE: Record<Code, string> = {
  [Code.Canceled]: '',
  [Code.Unknown]:
    'Nothing was done: numen did not answer. It may have stopped, and the window keeps trying.',
  [Code.InvalidArgument]: 'Nothing was done: numen could not make sense of what was asked.',
  [Code.DeadlineExceeded]: 'Nothing was done: numen took too long to answer. Try again.',
  [Code.NotFound]: 'What was asked for is not there.',
  [Code.AlreadyExists]: UNEXPECTED,
  [Code.PermissionDenied]: UNEXPECTED,
  [Code.ResourceExhausted]: UNEXPECTED,
  [Code.FailedPrecondition]:
    'Nothing was done: numen cannot do that as things stand. There may be no vault open, or nothing set to do it with.',
  [Code.Aborted]: UNEXPECTED,
  [Code.OutOfRange]: UNEXPECTED,
  [Code.Unimplemented]: 'This installation of numen cannot do that.',
  [Code.Internal]: UNEXPECTED,
  [Code.Unavailable]: 'Nothing was done: numen is not answering just now. Try again in a moment.',
  [Code.DataLoss]: UNEXPECTED,
  [Code.Unauthenticated]: UNEXPECTED,
}

/**
 * What a caught fault says to the person who was waiting for the answer.
 *
 * Anything at all can be thrown, so anything at all is taken. What the window
 * threw at itself is not the person's to read either, and lands on the same
 * sentence as an unexpected code.
 */
export const sentence = (thrown: unknown): string => TROUBLE[ConnectError.from(thrown).code]

/**
 * What each refusal the schema carries says, as the clause it is read in.
 *
 * Keyed by the schema itself, so a refusal added to it has no words until
 * someone writes them and does not compile until someone does.
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
export const refused = (refusal: Refusal | undefined): string =>
  refusal === undefined ? '' : REFUSED[refusal]
