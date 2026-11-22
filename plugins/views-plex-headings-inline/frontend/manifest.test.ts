/** The manifest of the plugin: the slot under the node it fills, its one setting and the one permission its reading needs. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  readonly id: string
  readonly name: string
  readonly permissions: readonly string[]
  readonly contributes: { readonly slots?: readonly Record<string, unknown>[] }
  readonly settings?: readonly { readonly key: string; readonly default: unknown }[]
}

describe('the manifest', () => {
  it('names the plugin apart from the dropdown', () => {
    expect([manifest.id, manifest.name]).toStrictEqual([
      'md.numen.views.plex.headings.inline',
      'Plex Headings Inline',
    ])
  })

  it('places its view under a node', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'plexGraph.nodeBelow', view: 'headings', title: 'Headings under a note' },
    ])
  })

  it('has one setting, the headings per note', () => {
    expect(manifest.settings?.map((one) => [one.key, one.default])).toStrictEqual([
      ['headingLimit', 6],
    ])
  })

  it('asks only to read the vault', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read'])
  })
})
