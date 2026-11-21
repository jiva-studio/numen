/** How many headings a node writes, as the plugin context holds the setting. */
import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import { readSettings } from './settings'

const createContext = (initialSettings = {}) =>
  createMockPluginContext({ pluginId: 'md.numen.views.plex.headings.inline', initialSettings })

describe('the setting headingLimit', () => {
  it('is six where nothing is set', () => {
    expect(readSettings(createContext()).headingLimit.value).toBe(6)
  })

  it('is clamped to one through twelve', () => {
    expect(readSettings(createContext({ headingLimit: 40 })).headingLimit.value).toBe(12)
    expect(readSettings(createContext({ headingLimit: 0 })).headingLimit.value).toBe(1)
  })

  it('follows a change of the setting', () => {
    const context = createContext()
    const settings = readSettings(context)

    context.setSetting('headingLimit', 3)

    expect(settings.headingLimit.value).toBe(3)
  })
})
