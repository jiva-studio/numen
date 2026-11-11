import { describe, expect, it } from 'vitest'
import { parseManifest } from './manifest'

describe('parseManifest', () => {
  it('reads a manifest and ignores the keys the schema does not name', () => {
    const answer = parseManifest(
      JSON.stringify({
        id: 'sample',
        name: 'Sample',
        version: '1.0.0',
        futureField: true,
        contributes: {
          feature: 'Samples',
          tabKinds: [{ kind: 'sample' }],
          commands: [{ id: 'sample.run', title: 'Run', permission: 'vault:manage' }],
        },
      }),
    )

    expect(answer.ok).toBe(true)
    if (!answer.ok) return
    expect(answer.value.contributes?.tabKinds[0]?.kind).toBe('sample')
    expect(answer.value.contributes?.commands[0]?.permission).toBe('vault:manage')
  })

  it('reports a value of the wrong type', () => {
    const answer = parseManifest('{"id": 5}')

    expect(answer.ok).toBe(false)
  })

  it('reports text that is not JSON', () => {
    expect(parseManifest('{').ok).toBe(false)
  })
})
