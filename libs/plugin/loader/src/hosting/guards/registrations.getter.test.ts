/**
 * Violation list of the check `registrations-read-once`: every test here fails until the plugin context
 * reads a registration once and hands the host that copy, as it does for a reader and a tab kind.
 */
import { createMockManifest } from '@numen/core-client/testing'
import type {
  CommandRegistration,
  SettingsViewRegistration,
} from '@numen/plugin-sdk'
import { describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { createTestContextHost } from '../../testing/contextHost'
import { createPluginContext } from '../context'

function createFlippingId(declared: string, other: string) {
  let reads = 0
  return () => {
    reads += 1
    return reads === 1 ? declared : other
  }
}

describe('RED UNTIL FIXED: a registration whose getter answers differently on each read', () => {
  it('registers a command under the id the manifest check read', () => {
    const registered = vi.fn((_pluginId: string, registration: CommandRegistration) => ({
      id: registration.id,
      dispose: vi.fn(),
    }))
    const host = createTestContextHost({ registration: { registerCommand: registered } })
    const manifest = createMockManifest('md.numen.sample', { commands: ['tree.open'] })
    const context = createPluginContext('md.numen.sample', host, manifest)
    const readId = createFlippingId('tree.open', 'tree.close')

    context.register.command({
      get id() {
        return readId()
      },
      handler: vi.fn(),
    })

    expect(registered.mock.results[0]?.value.id).toBe('tree.open')
  })

  it('registers a settings view under the id the manifest check read', () => {
    const registered = vi.fn((_pluginId: string, registration: SettingsViewRegistration) => ({
      id: registration.id,
      dispose: vi.fn(),
    }))
    const host = createTestContextHost({ registration: { registerSettingsView: registered } })
    const manifest = createMockManifest('md.numen.sample', { settingsViews: ['general'] })
    const context = createPluginContext('md.numen.sample', host, manifest)
    const readId = createFlippingId('general', 'advanced')

    context.register.settingsView({
      get id() {
        return readId()
      },
      component: defineComponent({}),
    })

    expect(registered.mock.results[0]?.value.id).toBe('general')
  })
})
