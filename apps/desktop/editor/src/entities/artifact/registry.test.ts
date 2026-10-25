import { describe, expect, it } from 'vitest'
import { createArtifactRegistry, useArtifactRegistry } from './registry'
import type { ArtifactDefinition } from './types'

const WORDS = {
  none: 'a',
  queued: 'b',
  running: 'c',
  stopped: 'd',
  done: 'e',
  empty: 'f',
  failed: 'g',
}

const definition = (id: string, wireKind?: number): ArtifactDefinition => ({
  id,
  ...(wireKind === undefined ? {} : { wireKind }),
  words: WORDS,
})

describe('artifact registry', () => {
  it('holds what is registered, by name and by schema number', () => {
    const registry = createArtifactRegistry()

    expect(registry.registerArtifact(definition('summary', 42))).toStrictEqual({
      ok: true,
      value: definition('summary', 42),
    })
    expect(registry.getArtifact('summary')?.id).toBe('summary')
    expect(registry.getArtifactByWireKind(42)?.id).toBe('summary')
    expect(registry.getArtifactWords()).toStrictEqual({ summary: WORDS })
  })

  it('refuses a name taken and a name empty', () => {
    const registry = createArtifactRegistry()
    registry.registerArtifact(definition('summary'))

    expect(registry.registerArtifact(definition('summary'))).toStrictEqual({
      ok: false,
      error: 'duplicate',
    })
    expect(registry.registerArtifact(definition(''))).toStrictEqual({ ok: false, error: 'invalid' })
    expect(registry.listArtifacts()).toHaveLength(1)
  })

  it('carries the built-in artifacts through the same registry', () => {
    const ids = useArtifactRegistry()
      .listArtifacts()
      .map((one) => one.id)

    expect(ids).toStrictEqual(['transcript', 'article', 'copy'])
  })

  it('offers a plugin artifact to the palette words', () => {
    useArtifactRegistry().registerArtifact(definition('plugin.notes'))

    expect(useArtifactRegistry().getArtifactWords()['plugin.notes']).toStrictEqual(WORDS)
  })
})
