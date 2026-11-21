/** The manifest of the plugin: the slot it fills and that it asks for nothing. */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly targets: readonly string[]
  readonly permissions: readonly string[]
  readonly backend?: unknown
  readonly contributes: { readonly slots?: readonly Record<string, unknown>[] }
}

describe('the manifest', () => {
  it('names the plugin Plex Navigation, for the editor window', () => {
    expect([manifest.id, manifest.name, manifest.targets]).toStrictEqual([
      'md.numen.views.plex.navigation',
      'Plex Navigation',
      ['desktop.editor'],
    ])
  })

  it('keeps the description within 64 characters', () => {
    expect(manifest.description.length).toBeLessThanOrEqual(64)
  })

  it('places the buttons at the top of the board', () => {
    expect(manifest.contributes.slots).toStrictEqual([
      { slot: 'plexGraph.boardTop', view: 'navigation', order: 10 },
    ])
  })

  it('asks for no permission and has no backend', () => {
    expect(manifest.permissions).toStrictEqual([])
    expect(manifest.backend).toBeUndefined()
  })
})
