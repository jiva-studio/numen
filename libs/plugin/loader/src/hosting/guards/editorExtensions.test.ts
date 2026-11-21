import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginParts } from '../context'
import { createTestContextHost } from '../../testing/contextHost'
import type { EditorExtensionRegistration } from '@numen/plugin-sdk'

function createRegisteringHost() {
  const registered = vi.fn((_id: string, _reg: EditorExtensionRegistration) => ({
    dispose: vi.fn(),
  }))
  const host = createTestContextHost({ views: { registerEditorExtension: registered } })
  return { host, registered }
}

describe('an editor extension registration', () => {
  it('fails for an id the manifest does not declare, naming the plugin and the id', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      editorExtensions: ['tasks'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.editorExtension?.({ id: 'other', extension: [] }),
    ).toThrowError('md.numen.sample')
    expect(registered).not.toHaveBeenCalled()
  })

  it('fails without vault:read even for a declared id', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', { editorExtensions: ['tasks'] })
    const context = createPluginParts('md.numen.sample', host, manifest)

    expect(() =>
      context.plugin.register.editorExtension?.({ id: 'tasks', extension: [] }),
    ).toThrow()
    expect(registered).not.toHaveBeenCalled()
  })

  it('hands a declared extension to the host under the plugin id', () => {
    const { host, registered } = createRegisteringHost()
    const manifest = createMockManifest('md.numen.sample', {
      permissions: ['vault:read'],
      editorExtensions: ['tasks'],
    })
    const context = createPluginParts('md.numen.sample', host, manifest)

    context.plugin.register.editorExtension?.({ id: 'tasks', extension: [] })

    expect(registered).toHaveBeenCalledWith('md.numen.sample', { id: 'tasks', extension: [] })
  })
})
