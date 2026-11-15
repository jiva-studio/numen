import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginParts } from '../context'
import { createTestContextHost } from '../../testing/contextHost'
import { STUB_COMPONENT } from '../../testing/stubComponent'

const PERMISSIONS = ['vault:read', 'vault:write']

function createRegisteringHost() {
  const registered = vi.fn(() => ({ dispose: vi.fn() }))
  const host = createTestContextHost({
    registration: {
      registerReader: registered,
      registerTabKind: registered,
      registerCommand: registered,
      registerSettingsView: registered,
    },
  })
  return { host, registered }
}

describe('a registration the manifest does not declare', () => {
  it('fails for a reader, naming the plugin, the kind and the id', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', { permissions: PERMISSIONS })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }),
    ).toThrowError(
      'Plugin "md.numen.sample" registers the reader "pdf", which its manifest does not declare in contributes.readers',
    )
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a tab kind of another id than the declared one', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: PERMISSIONS,
      tabKinds: ['deck'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.tabKind({ kind: 'cards', component: STUB_COMPONENT }),
    ).toThrowError('registers the tab kind "cards"')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a command, and for a shared command no handler declares', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: PERMISSIONS,
      commands: ['deck.open'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.command({ id: 'deck.close', handler: vi.fn() }),
    ).toThrowError('registers the command "deck.close"')
    expect(() =>
      context.plugin.register.command({ id: 'app:file.createText', handler: vi.fn() }),
    ).toThrowError('contributes.handlers')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a settings view', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', { permissions: PERMISSIONS })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.settingsView({ id: 'floor', component: STUB_COMPONENT }),
    ).toThrowError('registers the settings view "floor"')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for every registration of a plugin that has no manifest', () => {
    const { host } = createRegisteringHost()
    const context = createPluginParts('md.numen.sample', host, { permissions: PERMISSIONS })

    expect(() =>
      context.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }),
    ).toThrowError('does not declare')
  })
})

describe('a registration the manifest declares', () => {
  it('reaches the host', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: PERMISSIONS,
      readers: ['pdf'],
      tabKinds: ['deck'],
      commands: ['deck.open'],
      handlers: ['app:file.createText'],
      settingsViews: ['floor'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    context.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT })
    context.plugin.register.tabKind({ kind: 'deck', component: STUB_COMPONENT })
    context.plugin.register.command({ id: 'deck.open', handler: vi.fn() })
    context.plugin.register.command({ id: 'app:file.createText', handler: vi.fn() })
    context.plugin.register.settingsView({ id: 'floor', component: STUB_COMPONENT })

    expect(registered).toHaveBeenCalledTimes(5)
  })
})

describe('openExternal', () => {
  it('rejects for a plugin without link:open and does not open the address', async () => {
    const openExternal = vi.fn(() => Promise.resolve())
    const host = createTestContextHost({ info: { openExternal } })
    const context = createPluginParts(
      'md.numen.sample',
      host,
      createMockManifest('md.numen.sample', { permissions: PERMISSIONS }),
    )

    await expect(context.plugin.identity.openExternal('https://numen.md/')).rejects.toThrowError(
      'lacks required permission "link:open" for openExternal',
    )
    expect(openExternal).not.toHaveBeenCalled()
  })

  it('opens the address for a plugin that declares link:open', async () => {
    const openExternal = vi.fn(() => Promise.resolve())
    const host = createTestContextHost({ info: { openExternal } })
    const context = createPluginParts(
      'md.numen.sample',
      host,
      createMockManifest('md.numen.sample', { permissions: ['link:open'] }),
    )

    await expect(context.plugin.identity.openExternal('https://numen.md/')).resolves.toEqual({
      ok: true,
      value: null,
    })
    expect(openExternal).toHaveBeenCalledWith('https://numen.md/')
  })
})
