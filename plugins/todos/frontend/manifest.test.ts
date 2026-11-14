import { describe, expect, it } from 'vitest'
import { TODOS_VIEW } from './index'
import { SERVICE } from './api/todos'

interface Manifest {
  readonly id: string
  readonly version: string
  readonly targets: readonly string[]
  readonly permissions: readonly string[]
  readonly services: readonly string[]
  readonly contributes: {
    readonly tabKinds?: readonly unknown[]
    readonly commands?: readonly unknown[]
    readonly slots?: readonly { slot: string; view?: string }[]
  }
  readonly settings?: readonly { key: string }[]
}

const files = import.meta.glob('../manifest.json', { eager: true, import: 'default' }) as Record<
  string,
  Manifest
>
const packages = import.meta.glob('../package.json', { eager: true, import: 'default' }) as Record<
  string,
  { version: string }
>
const manifest = Object.values(files)[0] as Manifest

describe('the manifest of the todos plugin', () => {
  it('declares the editor target, the read and write permissions and the service the frontend calls', () => {
    expect(manifest.targets).toEqual(['desktop.editor'])
    expect(manifest.permissions).toEqual(['vault:read', 'vault:write'])
    expect(manifest.services).toEqual([SERVICE])
    expect(manifest.id).toBe(SERVICE)
  })

  it('declares the view the frontend registers in the title bar', () => {
    expect(manifest.contributes.slots?.map((one) => [one.slot, one.view])).toEqual([
      ['titleBar.status', TODOS_VIEW],
    ])
  })

  it('declares no tab kind, no command and the ignored folders setting only', () => {
    expect(manifest.contributes.tabKinds ?? []).toEqual([])
    expect(manifest.contributes.commands ?? []).toEqual([])
    expect(manifest.settings?.map((one) => one.key)).toEqual(['ignoredFolders'])
  })

  it('has the version of the package', () => {
    expect(Object.values(packages)[0]?.version).toBe(manifest.version)
  })
})
