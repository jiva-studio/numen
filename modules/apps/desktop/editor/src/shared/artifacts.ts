/**
 * The runs a model makes over one file of the vault: what a file carries
 * already, beginning one, and taking one away.
 *
 * A build that cannot do a run at all says so where it is asked, and the window
 * offers it nowhere afterwards.
 */
import { Code, createClient } from '@connectrpc/connect'
import type { ConnectError } from '@connectrpc/connect'
import {
  ArtifactKind as Kinds,
  ArtifactService,
  State as States,
} from '@numen/protocol'
import { namesOf, transport } from '@numen/wire'

/** What a model has made from the files of the vault. */
const artifacts = createClient(ArtifactService, transport)

/**
 * What a model makes from one file of the vault, asked for by name. Which model
 * does the work follows from the file, so the window names the artifact and
 * never the producer.
 */
export const running: ArtifactRunner = {
  carries: async (path) => {
    const answer = await artifacts.listArtifacts({ path })
    const held: Record<string, ArtifactState> = {}
    for (const one of answer.artifacts) {
      const of = drawn[one.kind]
      if (of) held[of] = reached(one.state)
    }
    return held
  },
  makes: async (path, of) => {
    try {
      const answer = await artifacts.createArtifact({ path, kind: asking[of] })
      return {
        able: true,
        of,
        made: reached(answer.artifact?.state),
        error: answer.artifact?.error ?? '',
      }
    } catch (error) {
      // A build that cannot make it at all says so, and it is offered nowhere
      // from then on.
      if (Code.Unimplemented === (error as ConnectError).code) return { able: false }
      throw error
    }
  },
  corrects: async (path) => {
    // Which text is put right follows from the file: a recording carries a
    // transcript and a scan carries a reading.
    const held = await running.carries(path)
    const of: Artifact = held.transcript === undefined ? 'ocr.corrected' : 'transcript.corrected'
    return running.makes(path, of)
  },
  fetches: async (path) => {
    // Which of the two the text at an address is, is the vault's to say: it
    // knows the address, and this asks for the one it says the note carries.
    const held = await running.carries(path)
    const of: Artifact = held.transcript === undefined ? 'article' : 'transcript'
    return running.makes(path, of)
  },
  deletesTranscript: (path) => taken(path, Kinds.TRANSCRIPT),
  deletesCopy: (path) => taken(path, Kinds.COPY),
}

/** One of what a file carries, taken away, and whether this build can do it. */
const taken = async (path: string, kind: Kinds): Promise<boolean> => {
  try {
    await artifacts.deleteArtifact({ path, kind })
    return true
  } catch (error) {
    if (Code.Unimplemented === (error as ConnectError).code) return false
    throw error
  }
}

/**
 * What each artifact the schema names is called in the window. Keyed by the
 * schema, so an artifact added to it has to be given a word here before this
 * compiles.
 */
const drawn: Readonly<Record<Kinds, Artifact | null>> = {
  [Kinds.UNSPECIFIED]: null,
  [Kinds.OCR]: 'ocr',
  [Kinds.OCR_CORRECTED]: 'ocr.corrected',
  [Kinds.TRANSCRIPT]: 'transcript',
  [Kinds.TRANSCRIPT_CORRECTED]: 'transcript.corrected',
  [Kinds.ARTICLE]: 'article',
  [Kinds.COPY]: 'copy',
}

/** And back, for asking for one. */
const asking = namesOf<Artifact, Kinds>(drawn)

/** What has become of an artifact, in the words the window uses. */
const become: Record<States, ArtifactState> = {
  [States.UNSPECIFIED]: 'none',
  [States.NONE]: 'none',
  [States.QUEUED]: 'queued',
  [States.RUNNING]: 'running',
  [States.STOPPED]: 'stopped',
  [States.DONE]: 'done',
  [States.EMPTY]: 'empty',
  [States.FAILED]: 'failed',
}

/** A state this window has no word for is an artifact nothing has made. */
const reached = (state: States | undefined): ArtifactState =>
  (state === undefined ? undefined : become[state]) ?? 'none'

/**
 * What is made from one file of the vault, by what it is: text with the place
 * on the page each word stands at, text with the times it was said at, either
 * of those put right, the prose a page is written around, and the bytes of a
 * video kept to be played.
 *
 * What made it is another question. A transcript is a transcript whether a
 * model here wrote it down or a site published it with a video.
 */
export type Artifact =
  | 'ocr'
  | 'ocr.corrected'
  | 'transcript'
  | 'transcript.corrected'
  | 'article'
  | 'copy'

/**
 * What has become of one artifact: nothing has been made, a run over it waits
 * its turn behind another, a run is writing it now, a run stopped part way and
 * what it reached is on disk, the whole of it stands, a run found nothing to
 * write down, or a run could not read the file at all.
 *
 * The last two are what a run answered, and asking again gets the same until
 * the artifact is taken away.
 */
export type ArtifactState =
  | 'none'
  | 'queued'
  | 'running'
  | 'stopped'
  | 'done'
  | 'empty'
  | 'failed'

/**
 * What a file carries, and what has become of each. Partial because which
 * artifacts a file carries follows from the file: a scan carries no
 * transcript, and a recording carries no text read.
 */
export type ArtifactStates = Partial<Record<Artifact, ArtifactState>>

/**
 * What asking for an artifact to be made answered: which artifact, what it now
 * is, and what a run said about a file it could not read. A build that cannot
 * make it at all answers nothing else, and the run is offered nowhere after
 * that.
 */
export type Outcome =
  | { readonly able: false }
  | {
      readonly able: true
      readonly of: Artifact
      readonly made: ArtifactState
      readonly error: string
    }

/** What a person asks be made from one file of the vault, and taken away. */
export interface ArtifactRunner {
  /**
   * What the file at a path carries. It is asked before anything is offered
   * over the file, so a book that has been read is not offered to be read
   * again.
   */
  carries(path: string): Promise<ArtifactStates>
  /** One artifact asked for, and what came of asking. */
  makes(path: string, of: Artifact): Promise<Outcome>
  /**
   * The text of a file put right by a proofreader, asked for by which text
   * that is: a recording carries a transcript and a scan carries a reading.
   */
  corrects(path: string): Promise<Outcome>
  /**
   * The text at the address a note points at, asked for by what that text is:
   * a video is a transcript and every other page is an article, and which of
   * them this note carries is what the vault answers.
   */
  fetches(path: string): Promise<Outcome>
  /**
   * The transcript of a recording taken away, with everything cut from it, and
   * whether this build can do it at all. The recording is left saying nothing,
   * and it is offered to be transcribed again.
   */
  deletesTranscript(path: string): Promise<boolean>
  /**
   * The copy fetched for a url taken off this disk, and whether this build can
   * do it at all. The url stands as it was, pointing where it points, and what
   * is there is framed again.
   */
  deletesCopy(path: string): Promise<boolean>
}
