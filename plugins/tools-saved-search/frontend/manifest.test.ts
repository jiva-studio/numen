/**
 * The tab kind the manifest declares for saved search notes is the one the frontend registers.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { WorkspaceHost } from '@numen/plugin-sdk'
import { createMockHostServices, createMockPluginContext } from '@numen/plugin-sdk/testing'
import { activate } from './index'

interface TabKind {
  readonly kind: string
  readonly title: string
  readonly icon: string
  readonly when: string
  readonly rank?: string
}

interface Manifest {
  readonly contributes: {
    readonly tabKinds?: readonly TabKind[]
    readonly commands?: readonly { readonly id: string }[]
    readonly slots?: readonly object[]
  }
  readonly frontend?: object
}

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as Manifest

describe('the manifest of the saved search frontend', () => {
  it('declares the kind it registers', () => {
    const context = createMockPluginContext({ pluginId: 'md.numen.tools.saved.search' })
    const workspace = { files: { openAt: () => undefined } } as unknown as WorkspaceHost

    activate(context, createMockHostServices({ workspace }))

    expect(manifest.contributes.tabKinds?.map((one) => one.kind)).toStrictEqual(
      context.getRegisteredTabKinds().map((one) => one.kind),
    )
  })

  it('opens notes of type search in the saved search tab', () => {
    expect(manifest.contributes.tabKinds).toStrictEqual([
      {
        kind: 'saved-search',
        title: 'Saved Search',
        icon: 'search',
        when: "target.noteType == 'search'",
      },
    ])
  })

  it('offers creating a saved search in the palette and the New menu of a folder', () => {
    expect(manifest.contributes.commands?.map((one) => one.id)).toStrictEqual(['search.create'])
    expect(manifest.contributes.slots).toStrictEqual([
      {
        slot: 'fileTree.new',
        command: 'search.create',
        group: '2-search',
        order: 10,
        when: 'target.isFolder',
      },
    ])
  })

  it('declares a frontend', () => {
    expect(manifest.frontend).toBeDefined()
  })
})
