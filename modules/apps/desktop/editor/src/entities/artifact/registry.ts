/**
 * Which artifacts the window knows. The built-in ones and those a plugin adds
 * stand in one registry, and a caller asks it for words and for schema numbers.
 */
import { computed, shallowReactive } from 'vue'
import { createBuiltInArtifacts } from './builtins'
import type { Artifact, ArtifactDefinition, ArtifactRegistryError, ArtifactState } from './types'

/** The artifacts registered now, each under its name. */
export interface ArtifactRegistry {
  /** Adds one artifact. A name already taken, or no name at all, is refused. */
  registerArtifact(
    definition: ArtifactDefinition,
  ): { ok: true; value: ArtifactDefinition } | { ok: false; error: ArtifactRegistryError }
  getArtifact(id: Artifact): ArtifactDefinition | undefined
  getArtifactByWireKind(wireKind: number): ArtifactDefinition | undefined
  /** Every artifact registered, in the order they were registered. */
  listArtifacts(): readonly ArtifactDefinition[]
  /** The words of every artifact, each by its name. */
  getArtifactWords(): Record<Artifact, Record<ArtifactState, string>>
}

/** A registry holding nothing but what is registered into it afterwards. */
export const createArtifactRegistry = (): ArtifactRegistry => {
  const byId = shallowReactive(new Map<Artifact, ArtifactDefinition>())
  return {
    registerArtifact: (definition) => {
      if (definition.id === '') return { ok: false, error: 'invalid' }
      if (byId.has(definition.id)) return { ok: false, error: 'duplicate' }
      byId.set(definition.id, definition)
      return { ok: true, value: definition }
    },
    getArtifact: (id) => byId.get(id),
    getArtifactByWireKind: (wireKind) =>
      [...byId.values()].find((one) => one.wireKind === wireKind),
    listArtifacts: () => [...byId.values()],
    getArtifactWords: () =>
      Object.fromEntries([...byId.values()].map((one) => [one.id, { ...one.words }])),
  }
}

const shared = createArtifactRegistry()
for (const one of createBuiltInArtifacts()) shared.registerArtifact(one)

/** The registry of this window, with the built-in artifacts already in it. */
export const useArtifactRegistry = () => ({
  ...shared,
  artifacts: computed(() => shared.listArtifacts()),
  words: computed(() => shared.getArtifactWords()),
})
