/// <reference types="node" />
/**
 * The manifest of `md.numen.tools.logs`: declares the `logs` tab kind, the `logs.open` command,
 * `logs:read` and `vault:read` permissions, and `isOffByDefault: true`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { LOGS_KIND, OPEN_LOGS_COMMAND } from './index'

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as {
  readonly id: string
  readonly targets: readonly string[]
  readonly permissions: readonly string[]
  readonly contributes: {
    readonly tabKinds: readonly { readonly kind: string; readonly title: string }[]
    readonly commands: readonly { readonly id: string; readonly title: string }[]
  }
  readonly isOffByDefault?: boolean
  readonly backend?: unknown
}

describe('the manifest', () => {
  it('declares the logs tab kind and the logs.open command in the editor', () => {
    expect(manifest.id).toBe('md.numen.tools.logs')
    expect(manifest.targets).toStrictEqual(['desktop.editor'])
    expect(manifest.contributes.tabKinds.map((one) => one.kind)).toStrictEqual([LOGS_KIND])
    expect(manifest.contributes.commands.map((one) => one.id)).toStrictEqual([OPEN_LOGS_COMMAND])
  })

  it('requests logs:read and vault:read and is off by default', () => {
    expect(manifest.permissions).toStrictEqual(['logs:read', 'vault:read'])
    expect(manifest.isOffByDefault).toBe(true)
    expect(manifest.backend).toBeUndefined()
  })
})
