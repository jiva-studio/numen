import { describe, expect, it } from 'vitest'
import { getUnavailableView, type TabKindPlugin } from './unavailableView'

const notes: TabKindPlugin = {
  id: 'sample.notes',
  name: 'Notes',
  feature: 'Markdown notes',
  standing: 'disabled',
  incompatibility: null,
  kindTitle: '',
}

describe('getUnavailableView', () => {
  it('has no view while the catalog has not answered', () => {
    expect(getUnavailableView({ status: 'pending' }, false)).toBeUndefined()
  })

  it('names no plugin for a kind the catalog does not know', () => {
    expect(getUnavailableView({ status: 'missing' }, false)).toMatchObject({
      heading: 'Plugin unavailable',
      action: 'inspect',
      pluginId: undefined,
    })
  })

  it('offers to enable a plugin that is switched off, and says that it is enabling', () => {
    const found = { status: 'found', plugin: notes } as const

    expect(getUnavailableView(found, false)).toStrictEqual({
      heading: 'Plugin disabled',
      message: 'The plugin for Markdown notes is disabled.',
      action: 'enable',
      actionText: 'Enable Notes',
      pluginId: 'sample.notes',
    })
    expect(getUnavailableView(found, true)?.actionText).toBe('Enabling Notes…')
  })

  it('states why a plugin cannot run, whatever else it is', () => {
    const incompatible = { ...notes, incompatibility: 'needs 2.0' }

    expect(getUnavailableView({ status: 'found', plugin: incompatible }, false)).toStrictEqual({
      heading: 'Plugin incompatible',
      message: 'Notes cannot run in this application: needs 2.0',
      action: 'inspect',
      actionText: 'Open Settings',
      pluginId: 'sample.notes',
    })
  })

  it('offers to install a plugin that a source offers', () => {
    const uninstalled = { ...notes, feature: 'Notes', standing: 'offered' } as const

    expect(getUnavailableView({ status: 'found', plugin: uninstalled }, false)).toStrictEqual({
      heading: 'Plugin not installed',
      message: 'The plugin for Notes is not installed.',
      action: 'install',
      actionText: 'Install Notes',
      pluginId: 'sample.notes',
    })
  })

  it('sends the person to the settings for a plugin that is installed and not running', () => {
    const installed = { ...notes, standing: 'installed' } as const

    expect(getUnavailableView({ status: 'found', plugin: installed }, false)).toStrictEqual({
      heading: 'Plugin not running',
      message: 'The plugin for Markdown notes is not running.',
      action: 'inspect',
      actionText: 'Open Settings',
      pluginId: 'sample.notes',
    })
  })
})
