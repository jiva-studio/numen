/** What a model makes from a file of the vault: the runs, and what to call them. */
export { running } from './api/artifacts'
export { FETCHED } from './words'
export { createArtifactRegistry, useArtifactRegistry } from './registry'
export type { ArtifactRegistry } from './registry'
export type {
  Artifact,
  ArtifactDefinition,
  ArtifactRegistryError,
  BuiltInArtifact,
  ArtifactCorrector,
  ArtifactDeleter,
  ArtifactFetcher,
  ArtifactInspector,
  ArtifactProducer,
  ArtifactRunner,
  ArtifactState,
  ArtifactStates,
  Outcome,
} from './types'
