import { describe, expect, it, vi } from 'vitest'
import { CONVERSATION, NOTE_EDITOR } from '@numen/plugin-sdk'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginParts } from '../context'
import {
  requireDeclaredCommand,
  requireDeclaredDropTypes,
  requireDeclaredReader,
  requireDeclaredSettingsView,
  requireDeclaredTabKind,
} from './declarations'
import type { PermissionTarget } from './permission'
import { requireMountAccess, requireViewRegistration } from './viewPermissions'
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
    expect(() =>
      requireDeclaredReader(
        {
          pluginId: 'md.numen.sample',
          manifest: createMockManifest('md.numen.sample', { readers: ['epub'] }),
        },
        'pdf',
      ),
    ).toThrowError('in contributes.readers')
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
    ).toThrowError(
      'Plugin "md.numen.sample" registers the tab kind "cards", which its manifest does not declare in contributes.tabKinds',
    )
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
    ).toThrowError(
      'Plugin "md.numen.sample" registers the command "deck.close", which its manifest does not declare in contributes.commands',
    )
    expect(() =>
      context.plugin.register.command({ id: 'app:file.createText', handler: vi.fn() }),
    ).toThrowError('contributes.handlers')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for a settings view', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: PERMISSIONS,
      settingsViews: ['other'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.settingsView({ id: 'floor', component: STUB_COMPONENT }),
    ).toThrowError(
      'Plugin "md.numen.sample" registers the settings view "floor", which its manifest does not declare in settingsViews',
    )
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails for every registration of a plugin that has no manifest or missing contribution lists', () => {
    const { host } = createRegisteringHost()
    const context = createPluginParts('md.numen.sample', host, { permissions: PERMISSIONS })

    expect(() =>
      context.plugin.register.reader({ id: 'pdf', component: STUB_COMPONENT }),
    ).toThrowError('does not declare')

    const emptyContributes = createMockManifest('p')
    if (emptyContributes.contributes) {
      emptyContributes.contributes.viewCapabilities = undefined as never
      emptyContributes.contributes.readers = undefined as never
      emptyContributes.contributes.tabKinds = undefined as never
      emptyContributes.contributes.commands = undefined as never
      emptyContributes.contributes.handlers = undefined as never
      emptyContributes.contributes.dropHandlers = undefined as never
    }
    const noContributes = createMockManifest('p')
    noContributes.contributes = undefined
    const targets: PermissionTarget[] = [
      { pluginId: 'p' },
      { pluginId: 'p', manifest: {} },
      { pluginId: 'p', manifest: noContributes },
      { pluginId: 'p', manifest: emptyContributes },
    ]
    for (const target of targets) {
      expect(() => requireDeclaredReader(target, 'pdf')).toThrowError('contributes.readers')
      expect(() => requireDeclaredTabKind(target, 'deck')).toThrowError('contributes.tabKinds')
      expect(() => requireDeclaredCommand(target, 'cmd', false)).toThrowError(
        'contributes.commands',
      )
      expect(() => requireDeclaredCommand(target, 'app:file.createText', true)).toThrowError(
        'contributes.handlers',
      )
      expect(() => requireDeclaredSettingsView(target, 'floor')).toThrowError('settingsViews')
      expect(() => requireDeclaredDropTypes(target, ['text/uri-list'])).toThrowError(
        'contributes.dropHandlers',
      )
      expect(() => requireViewRegistration(CONVERSATION, target)).toThrowError(
        'contributes.viewCapabilities',
      )
    }

    expect(requireMountAccess(CONVERSATION, { pluginId: 'p' }).permissions).toStrictEqual([])
    expect(() => requireMountAccess(NOTE_EDITOR, { pluginId: 'p' })).toThrowError('for mountView')
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
  it('answers permissionDenied for a plugin without link:open and does not open the address', async () => {
    const openExternal = vi.fn(() => Promise.resolve())
    const host = createTestContextHost({ ui: { openExternal } })
    const context = createPluginParts(
      'md.numen.sample',
      host,
      createMockManifest('md.numen.sample', { permissions: PERMISSIONS }),
    )

    await expect(context.plugin.identity.openExternal('https://numen.md/')).resolves.toEqual({
      ok: false,
      error: {
        error: 'permissionDenied',
        message: 'Plugin "md.numen.sample" lacks required permission "link:open" for openExternal',
      },
    })
    expect(openExternal).not.toHaveBeenCalled()
  })

  it('opens the address for a plugin that declares link:open', async () => {
    const openExternal = vi.fn(() => Promise.resolve())
    const host = createTestContextHost({ ui: { openExternal } })
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
