/** The artifacts the window registers itself, by the schema number of each. */
import { ArtifactKind as Kinds } from '@numen/protocol'
import type { ArtifactDefinition, BuiltInArtifact } from './types'
import { BUILT_IN_WORDS } from './words'

const WIRE_KINDS: Readonly<Record<BuiltInArtifact, Kinds>> = {
  transcript: Kinds.TRANSCRIPT,
  article: Kinds.ARTICLE,
  copy: Kinds.COPY,
}

export const createBuiltInArtifacts = (): readonly ArtifactDefinition[] =>
  (Object.keys(WIRE_KINDS) as BuiltInArtifact[]).map((id) => ({
    id,
    wireKind: WIRE_KINDS[id],
    words: BUILT_IN_WORDS[id],
  }))
