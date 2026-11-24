/**
 * The manifest of the sync plugin declares what the notes sync needs: the permissions to read and write
 * the vault, the service of the backend, the event that wakes it and the setting of its interval.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

interface Setting {
  readonly key: string
  readonly type: string
  readonly default?: unknown
  readonly min?: number
  readonly description?: string
}

interface Manifest {
  readonly permissions: readonly string[]
  readonly services?: readonly string[]
  readonly activationEvents?: readonly string[]
  readonly settings: readonly Setting[]
  readonly settingsViews?: readonly {
    readonly id: string
    readonly title: string
    readonly page?: string
  }[]
  readonly contributes?: Record<string, unknown>
}

const manifest = JSON.parse(
  readFileSync(resolve(import.meta.dirname, '../manifest.json'), 'utf8'),
) as Manifest

describe('the manifest of the sync plugin', () => {
  it('asks to read and write the vault besides the identity', () => {
    expect(manifest.permissions).toStrictEqual(
      expect.arrayContaining(['identity', 'identity:link', 'vault:read', 'vault:write']),
    )
  })

  it('serves its own service and wakes on a change of any file', () => {
    expect(manifest.services).toStrictEqual(['md.numen.tools.sync'])
    expect(manifest.activationEvents).toStrictEqual(['onFilesChanged'])
  })

  it('has a setting of how often this device asks the server for changes', () => {
    const interval = manifest.settings.find((one) => one.key === 'syncIntervalSeconds')

    expect(interval).toMatchObject({
      type: 'number',
      default: 30,
      min: 1,
      description: 'How often this device asks the server for changes',
    })
  })

  it('has a setting of how long a removed file is watched before the removal is sent', () => {
    const delay = manifest.settings.find((one) => one.key === 'removalDelaySeconds')

    expect(delay).toMatchObject({
      type: 'number',
      default: 60,
      min: 1,
      description: 'How long a removed file is watched before the removal is sent',
    })
  })

  it('keeps the setting of the server', () => {
    expect(manifest.settings.map((one) => one.key)).toContain('server')
  })

  it('declares the views of the synced vaults and the shared folders, and the subscription view of the page of the identity', () => {
    expect(manifest.settingsViews).toStrictEqual([
      { id: 'vaults', title: 'Synced vaults' },
      { id: 'shares', title: 'Shared folders' },
      { id: 'subscription', title: 'Subscription', page: 'identity' },
    ])
  })

  it('offers "Sync this vault" on the root and "Share..." on a folder of the file tree, and has a setting for the folder of shared vaults', () => {
    const contributes = manifest.contributes as {
      slots: readonly Record<string, string>[]
      commands: readonly Record<string, string>[]
    }

    expect(contributes.commands).toContainEqual(
      expect.objectContaining({ id: 'share.open', title: 'Share...' }),
    )
    expect(contributes.slots).toContainEqual(
      expect.objectContaining({
        slot: 'fileTree.context',
        command: 'share.open',
        when: 'target.isFolder && !target.isRoot',
      }),
    )
    expect(contributes.commands).toContainEqual(
      expect.objectContaining({ id: 'vault.sync', title: 'Sync this vault' }),
    )
    expect(contributes.slots).toContainEqual(
      expect.objectContaining({
        slot: 'fileTree.context',
        command: 'vault.sync',
        when: 'target.isRoot',
      }),
    )
    expect(contributes.commands).toContainEqual(
      expect.objectContaining({ id: 'share.accept', title: 'Accept a shared folder...' }),
    )
    expect(contributes).not.toHaveProperty('tabKinds')
    expect(manifest.settings.find((one) => one.key === 'receivedFolder')).toMatchObject({
      type: 'string',
      default: '',
    })
  })

  it('asks to open links of the system browser', () => {
    expect(manifest.permissions).toContain('link:open')
  })

  it('declares no license of its own', () => {
    expect(manifest.contributes).not.toHaveProperty('licenses')
  })
})
