/** The settings of the names, as the plugin context holds them. */
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { readSettings } from './settings'

const createContext = (initialSettings = {}) =>
  createMockPluginContext({ pluginId: 'md.numen.views.plex.related.names', initialSettings })

describe('the setting showRelationArrows', () => {
  it('is on where nothing is set', () => {
    expect(readSettings(createContext()).showRelationArrows.value).toBe(true)
  })

  it('is off when it is set to false', () => {
    expect(
      readSettings(createContext({ showRelationArrows: false })).showRelationArrows.value,
    ).toBe(false)
  })

  it('is on for a value that is not a yes or no', () => {
    expect(readSettings(createContext({ showRelationArrows: 'no' })).showRelationArrows.value).toBe(
      true,
    )
  })

  it('follows a change of the setting', () => {
    const context = createContext()
    const settings = readSettings(context)

    context.setSetting('showRelationArrows', false)
    expect(settings.showRelationArrows.value).toBe(false)

    context.setSetting('showRelationArrows', true)
    expect(settings.showRelationArrows.value).toBe(true)
  })
})
