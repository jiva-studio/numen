/**
 * The runs a model makes over one file of the vault: what a file carries
 * already, beginning one, and taking one away.
 *
 * A build that cannot do a run at all says so where it is asked, and the window
 * offers it nowhere afterwards.
 */
import { Code } from '@connectrpc/connect'
import type { ConnectError } from '@connectrpc/connect'
import { ArtifactKind as Kinds, State as States } from '@numen/protocol'
import { namesOf } from '@numen/wire'
import { artifacts } from '@/shared/clients'
import type { Artifact, ArtifactRunner, ArtifactState } from '../types'

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
      if (of) held[of] = parseState(one.state)
    }
    return held
  },
  createArtifact: async (path, of) => {
    try {
      const answer = await artifacts.createArtifact({ path, kind: asking[of] })
      return {
        able: true,
        of,
        made: parseState(answer.artifact?.state),
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
  deleteTranscript: (path) => deleteArtifact(path, Kinds.TRANSCRIPT),
  deleteCopy: (path) => deleteArtifact(path, Kinds.COPY),
}

/** One of what a file carries, taken away, and whether this build can do it. */
const deleteArtifact = async (path: string, kind: Kinds): Promise<boolean> => {
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
const parseState = (state: States | undefined): ArtifactState =>
  (state === undefined ? undefined : become[state]) ?? 'none'
