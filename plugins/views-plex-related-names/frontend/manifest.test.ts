/**
 * The manifest of the plugin: what it places in the plex, the settings it owns and the one
 * permission its reading needs.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

interface Setting {
  readonly key: string
  readonly type: string
  readonly title: string
  readonly description?: string
  readonly default: unknown
  readonly min?: number
  readonly max?: number
  readonly step?: number
  readonly palette?: boolean
  readonly icon?: string
}

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  readonly id: string
  readonly name: string
  readonly targets: readonly string[]
  readonly permissions: readonly string[]
  readonly activationEvents?: readonly string[]
  readonly backend?: unknown
  readonly frontend?: unknown
  readonly contributes: {
    readonly commands?: readonly unknown[]
    readonly slots?: readonly Record<string, unknown>[]
  }
  readonly settings?: readonly Setting[]
}

describe('the manifest', () => {
  it('names the plugin Plex Related Names, for the editor window', () => {
    expect([manifest.id, manifest.name, manifest.targets]).toStrictEqual([
      'md.numen.views.plex.related.names',
      'Plex Related Names',
      ['desktop.editor'],
    ])
  })

  it('has a frontend and no backend', () => {
    expect(manifest.frontend).toBeDefined()
    expect(manifest.backend).toBeUndefined()
  })

  it('starts with the window, with no event to wait for', () => {
    expect(manifest.activationEvents).toBeUndefined()
  })

  it('places the parents above a node of the plex and the related notes under it', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'plexGraph.nodeAbove', view: 'parents', title: 'Parent notes' },
      { slot: 'plexGraph.nodeBelow', view: 'related', title: 'Related notes' },
    ])
  })

  it('declares no command, since choosing a name runs the command of the window', () => {
    expect(manifest.contributes.commands ?? []).toStrictEqual([])
  })

  it('asks only to read the vault', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read'])
  })
})

describe('the settings of the names', () => {
  const find = (key: string) => (manifest.settings ?? []).find((one) => one.key === key)

  it('are namesPerNote and showRelationArrows, since the names are shown while the plugin is enabled', () => {
    expect((manifest.settings ?? []).map((one) => one.key).sort()).toStrictEqual([
      'namesPerNote',
      'showRelationArrows',
    ])
  })

  it('offers Names per note from 1 to 12, six by default', () => {
    expect(find('namesPerNote')).toMatchObject({
      type: 'range',
      title: 'Names per note',
      default: 6,
      min: 1,
      max: 12,
      step: 1,
    })
  })

  it('offers Show the relation arrows, on by default, with a description of the arrows', () => {
    expect(find('showRelationArrows')).toMatchObject({
      type: 'boolean',
      title: 'Show the relation arrows',
      default: true,
    })
    expect(find('showRelationArrows')?.description).toMatch(/arrow/i)
  })
})
