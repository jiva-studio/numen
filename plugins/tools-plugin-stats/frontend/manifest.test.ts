/// <reference types="node" />
/**
 * The manifest: one view in the title bar of the editor, the one permission the list of running
 * plugins needs, and off until a person turns it on.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { PLUGINS_VIEW } from './index'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  readonly id: string
  readonly targets: readonly string[]
  readonly permissions: readonly string[]
  readonly contributes: {
    readonly slots: readonly { readonly slot: string; readonly view: string }[]
  }
  readonly isOffByDefault?: boolean
  readonly backend?: unknown
  readonly settings?: unknown
}

describe('the manifest', () => {
  it('declares the title bar view the frontend registers, in the editor', () => {
    expect(manifest.id).toBe('md.numen.tools.plugin.stats')
    expect(manifest.targets).toStrictEqual(['desktop.editor'])
    expect(manifest.contributes.slots.map((one) => [one.slot, one.view])).toStrictEqual([
      ['titleBar.status', PLUGINS_VIEW],
    ])
  })

  it('reads the running plugins and nothing of a vault', () => {
    expect(manifest.permissions).toStrictEqual(['plugins:read'])
  })

  it('is off by default and has no backend and no settings', () => {
    expect(manifest.isOffByDefault).toBe(true)
    expect(manifest.backend).toBeUndefined()
    expect(manifest.settings).toBeUndefined()
  })
})
