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
import type { Artifact as ArtifactOf, ArtifactRunner, ArtifactState } from './core'

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
    const of: ArtifactOf = held.transcript === undefined ? 'ocr.corrected' : 'transcript.corrected'
    return running.makes(path, of)
  },
  fetches: async (path) => {
    // Which of the two the text at an address is, is the vault's to say: it
    // knows the address, and this asks for the one it says the note carries.
    const held = await running.carries(path)
    const of: ArtifactOf = held.transcript === undefined ? 'article' : 'transcript'
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
const drawn: Readonly<Record<Kinds, ArtifactOf | null>> = {
  [Kinds.UNSPECIFIED]: null,
  [Kinds.OCR]: 'ocr',
  [Kinds.OCR_CORRECTED]: 'ocr.corrected',
  [Kinds.TRANSCRIPT]: 'transcript',
  [Kinds.TRANSCRIPT_CORRECTED]: 'transcript.corrected',
  [Kinds.ARTICLE]: 'article',
  [Kinds.COPY]: 'copy',
}

/** And back, for asking for one. */
const asking = namesOf<ArtifactOf, Kinds>(drawn)

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
