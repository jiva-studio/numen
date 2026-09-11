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
  getArtifactStates: async (path) => {
    const answer = await artifacts.listArtifacts({ path })
    const held: Record<string, ArtifactState> = {}
    for (const one of answer.artifacts) {
      const of = drawn[one.kind]
      if (of) held[of] = reached(one.state)
    }
    return held
  },
  createArtifact: async (path, of) => {
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
  correctArtifact: async (path) => {
    // Which text is put right follows from the file: a recording carries a
    // transcript and a scan carries a reading.
    const held = await running.getArtifactStates(path)
    const of: Artifact = held.transcript === undefined ? 'ocr.corrected' : 'transcript.corrected'
    return running.createArtifact(path, of)
  },
  fetchArtifact: async (path) => {
    // Which of the two the text at an address is, is the vault's to say: it
    // knows the address, and this asks for the one it says the note carries.
    const held = await running.getArtifactStates(path)
    const of: Artifact = held.transcript === undefined ? 'article' : 'transcript'
    return running.createArtifact(path, of)
  },
  deleteTranscript: (path) => taken(path, Kinds.TRANSCRIPT),
  deleteCopy: (path) => taken(path, Kinds.COPY),
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

/** Reads which artifacts a file currently carries. */
export interface ArtifactInspector {
  getArtifactStates(path: string): Promise<ArtifactStates>
}

/** Initiates creation/recognition of an artifact for a file. */
export interface ArtifactProducer {
  createArtifact(path: string, of: Artifact): Promise<Outcome>
}

/** Corrects a file's OCR reading or transcript. */
export interface ArtifactCorrector {
  correctArtifact(path: string): Promise<Outcome>
}

/** Fetches content (transcript or article) for a URL address. */
export interface ArtifactFetcher {
  fetchArtifact(path: string): Promise<Outcome>
}

/** Removes artifact data from disk. */
export interface ArtifactDeleter {
  deleteTranscript(path: string): Promise<boolean>
  deleteCopy(path: string): Promise<boolean>
}

/** Composed interface combining all artifact capabilities. */
export interface ArtifactRunner
  extends ArtifactInspector,
    ArtifactProducer,
    ArtifactCorrector,
    ArtifactFetcher,
    ArtifactDeleter {}
