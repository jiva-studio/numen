/**
 * The manifest of the plugin: what it places in the plex, the settings it owns and the one permission its
 * reading needs.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

interface Setting {
  readonly key: string
  readonly type: string
  readonly title: string
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
  it('names the plugin Plex Headings Dropdown, for the editor window', () => {
    expect([manifest.id, manifest.name, manifest.targets]).toStrictEqual([
      'md.numen.views.plex.headings.dropdown',
      'Plex Headings Dropdown',
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

  it('places its list across the whole of a node of the plex', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'plexGraph.node', view: 'headings', title: 'Headings list' },
    ])
  })

  it('declares no command, since choosing a heading opens a note through the window', () => {
    expect(manifest.contributes.commands ?? []).toStrictEqual([])
  })

  it('asks only to read the vault', () => {
    expect(manifest.permissions).toStrictEqual(['vault:read'])
  })
})

describe('the setting of the headings', () => {
  const find = (key: string) => (manifest.settings ?? []).find((one) => one.key === key)

  it('is headingLimit, the key the plex used', () => {
    expect((manifest.settings ?? []).map((one) => one.key)).toStrictEqual(['headingLimit'])
  })

  it('offers Headings per note from 1 to 12, six by default', () => {
    expect(find('headingLimit')).toMatchObject({
      type: 'range',
      title: 'Headings per note',
      default: 6,
      min: 1,
      max: 12,
      step: 1,
    })
  })
})
